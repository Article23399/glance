package feed

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

type HuyaChannel struct {
	RoomId       string
	Exists       bool
	Name         string
	AvatarUrl    string
	IsLive       bool
	LiveSince    time.Time
	Category     string
	CategorySlug string
	ViewersCount int
}

type HuyaChannels []HuyaChannel

func (channels HuyaChannels) SortByViewers() {
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].ViewersCount > channels[j].ViewersCount
	})
}

type huyaStreamMetadataResponse struct {
	RoomInfo *struct {
		LiveStatus int `json:"eLiveStatus"`
		LiveInfo   *struct {
			Nick         string `json:"sNick"`
			UserCount    int    `json:"lUserCount"`
			Avatar180    string `json:"sAvatar180"`
			Introduction string `json:"sIntroduction"`
			GameFullName string `json:"sGameFullName"`
			GameHostName string `json:"sGameHostName"`
			StartTime    int64  `json:"iStartTime"`
		} `json:"tLiveInfo"`
	} `json:"roomInfo"`
}

const kUserAgent = "Mozilla/5.0 (Linux; Android 11; Pixel 5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.91 Mobile Safari/537.36 Edg/117.0.0.0"

func fetchChannelFromHuyaTask(channel string) (HuyaChannel, error) {
	result := HuyaChannel{
		RoomId: strings.ToLower(channel),
	}

	request, _ := http.NewRequest("GET", fmt.Sprintf("https://m.huya.com/%s", channel), nil)
	request.Header.Add("user-agent", kUserAgent)

	response, err := textFromRequest(defaultClient, request)

	if err != nil {
		return result, err
	}

	re := regexp.MustCompile(`window\.HNF_GLOBAL_INIT.=.\{(.*?)\}.</script>`)
	text := re.FindStringSubmatch(response)[1]

	var streamMetadata huyaStreamMetadataResponse

	err = json.Unmarshal([]byte(fmt.Sprintf("{%s}", text)), &streamMetadata)

	if err != nil {
		return result, fmt.Errorf("failed to unmarshal stream metadata: %w", err)
	}

	result.Exists = true
	result.Name = streamMetadata.RoomInfo.LiveInfo.Introduction
	result.AvatarUrl = streamMetadata.RoomInfo.LiveInfo.Avatar180
	result.IsLive = streamMetadata.RoomInfo.LiveStatus == 2
	result.ViewersCount = streamMetadata.RoomInfo.LiveInfo.UserCount
	result.Category = streamMetadata.RoomInfo.LiveInfo.GameFullName
	result.CategorySlug = streamMetadata.RoomInfo.LiveInfo.GameHostName
	result.LiveSince = time.Unix(streamMetadata.RoomInfo.LiveInfo.StartTime, 0)

	return result, nil
}

func FetchChannelsFromHuya(channelLogins []string) (HuyaChannels, error) {
	result := make(HuyaChannels, 0, len(channelLogins))

	job := newJob(fetchChannelFromHuyaTask, channelLogins).withWorkers(10)
	channels, errs, err := workerPoolDo(job)

	if err != nil {
		return result, err
	}

	var failed int

	for i := range channels {
		if errs[i] != nil {
			failed++
			slog.Warn("failed to fetch huya channel", "channel", channelLogins[i], "error", errs[i])
			continue
		}

		result = append(result, channels[i])
	}

	if failed == len(channelLogins) {
		return result, ErrNoContent
	}

	if failed > 0 {
		return result, fmt.Errorf("%w: failed to fetch %d channels", ErrPartialContent, failed)
	}

	return result, nil
}
