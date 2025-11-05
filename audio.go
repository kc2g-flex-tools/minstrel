// Audio stream lifecycle and audio packet processing
// Extracted from flexclient.go on 2025-11-02

package main

import (
	"context"
	"fmt"

	"github.com/hb9fxq/flexlib-go/vita"
	"github.com/kc2g-flex-tools/flexclient"
)

func (rs *RadioState) playOpus(pkt flexclient.VitaPacket) {
	data := vita.ParseVitaOpus(pkt.Payload, pkt.Preamble)
	rs.Audio.Decode(data)
}

func (rs *RadioState) ToggleAudio(enable bool) {
	if enable {
		rs.FlexClient.SendCmd("stream create type=remote_audio_rx compression=opus")
		rs.FlexClient.SendCmd("stream create type=remote_audio_tx compression=opus")
		rs.Audio.Start()
		rs.Audio.StartTX(rs.FlexClient, &rs.TXAudioStream)
	} else {
		rs.FlexClient.SendCmd(fmt.Sprintf("stream remove 0x%08x", rs.RXAudioStream))
		rs.RXAudioStream = 0
		if rs.TXAudioStream != 0 {
			rs.FlexClient.SendCmd(fmt.Sprintf("stream remove 0x%08x", rs.TXAudioStream))
			rs.TXAudioStream = 0
		}
		rs.Audio.Pause()
		rs.Audio.StopTX()
	}
}

func (rs *RadioState) SetPTT(enable bool) {
	xmit := "0"
	if enable {
		xmit = "1"
	}
	rs.FlexClient.SendCmd(fmt.Sprintf("xmit %s", xmit))
}

func (rs *RadioState) SetVOX(enable bool) {
	value := "0"
	if enable {
		value = "1"
	}
	rs.FlexClient.TransmitSet(context.Background(), flexclient.Object{"vox_enable": value})
}

func (rs *RadioState) SetTransmitParam(key string, value int) {
	rs.FlexClient.TransmitSet(context.Background(), flexclient.Object{key: fmt.Sprintf("%d", value)})
}

func (rs *RadioState) SetAMCarrierLevel(level int) {
	rs.FlexClient.TransmitAMCarrierSet(context.Background(), fmt.Sprintf("%d", level))
}

func (rs *RadioState) SetMicLevel(level int) {
	rs.FlexClient.TransmitMicLevelSet(context.Background(), fmt.Sprintf("%d", level))
}
