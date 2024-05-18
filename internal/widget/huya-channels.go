package widget

import (
	"context"
	"html/template"
	"time"

	"github.com/glanceapp/glance/internal/assets"
	"github.com/glanceapp/glance/internal/feed"
)

type HuyaChannels struct {
	widgetBase      `yaml:",inline"`
	ChannelsRequest []string           `yaml:"channels"`
	Channels        []feed.HuyaChannel `yaml:"-"`
	CollapseAfter   int                `yaml:"collapse-after"`
}

func (widget *HuyaChannels) Initialize() error {
	widget.withTitle("Huya Channels").withCacheDuration(time.Minute * 10)

	if widget.CollapseAfter == 0 || widget.CollapseAfter < -1 {
		widget.CollapseAfter = 5
	}

	return nil
}

func (widget *HuyaChannels) Update(ctx context.Context) {
	channels, err := feed.FetchChannelsFromHuya(widget.ChannelsRequest)

	if !widget.canContinueUpdateAfterHandlingErr(err) {
		return
	}

	channels.SortByViewers()
	widget.Channels = channels
}

func (widget *HuyaChannels) Render() template.HTML {
	return widget.render(widget, assets.HuyaChannelsTemplate)
}
