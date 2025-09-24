package opus

/*
#cgo pkg-config: opus
#include <opus.h>
#include <stdlib.h>

int opus_encoder_set_bitrate(OpusEncoder *st, opus_int32 bitrate) {
    return opus_encoder_ctl(st, OPUS_SET_BITRATE_REQUEST, bitrate);
}

int opus_encoder_set_complexity(OpusEncoder *st, int complexity) {
	return opus_encoder_ctl(st, OPUS_SET_COMPLEXITY_REQUEST, complexity);
}

int opus_encoder_set_vbr(OpusEncoder *st, int vbr) {
	return opus_encoder_ctl(st, OPUS_SET_VBR_REQUEST, vbr);
}

*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

const (
	ApplicationVoIP               = 2048
	ApplicationAudio              = 2049
	ApplicationRestrictedLowDelay = 2051
)

type Encoder struct {
	encoder  *C.OpusEncoder
	channels int
}

func NewEncoder(sampleRate, channels, application int) (*Encoder, error) {
	var err C.int

	enc := C.opus_encoder_create(
		C.opus_int32(sampleRate),
		C.int(channels),
		C.int(application),
		&err,
	)

	if err != C.OPUS_OK {
		return nil, errors.New("failed to create opus encoder")
	}

	return &Encoder{encoder: enc, channels: channels}, nil
}

func (e *Encoder) Encode(pcm []int16) ([]byte, error) {
	maxBytes := 1276
	data := make([]byte, maxBytes)

	ret := C.opus_encode(
		e.encoder,
		(*C.opus_int16)(unsafe.Pointer(&pcm[0])),
		C.int(len(pcm)/e.channels),
		(*C.uchar)(unsafe.Pointer(&data[0])),
		C.opus_int32(maxBytes),
	)

	if ret < 0 {
		return nil, fmt.Errorf("failed to encode %d samples on opus encoder %p: %d", len(pcm)/e.channels, e.encoder, ret)
	}

	return data[:ret], nil
}

func (e *Encoder) EncodeFloat(pcm []float32) ([]byte, error) {
	maxBytes := 1276
	data := make([]byte, maxBytes)
	numSamples := len(pcm) / 4

	ret := C.opus_encode_float(
		e.encoder,
		(*C.float)(unsafe.Pointer(&pcm[0])),
		C.int(numSamples/e.channels),
		(*C.uchar)(unsafe.Pointer(&data[0])),
		C.opus_int32(maxBytes),
	)

	if ret < 0 {
		return nil, fmt.Errorf("failed to encode %d samples on opus encoder %p: %d", numSamples/e.channels, e.encoder, ret)
	}

	return data[:ret], nil
}

func (e *Encoder) EncodeFloatRaw(pcm []byte) ([]byte, error) {
	maxBytes := 1276
	data := make([]byte, maxBytes)

	ret := C.opus_encode_float(
		e.encoder,
		(*C.float)(unsafe.Pointer(&pcm[0])),
		C.int(len(pcm)/(e.channels*4)),
		(*C.uchar)(unsafe.Pointer(&data[0])),
		C.opus_int32(maxBytes),
	)

	if ret < 0 {
		return nil, fmt.Errorf("failed to encode %d samples on opus encoder %p: %d", len(pcm)/e.channels, e.encoder, ret)
	}

	return data[:ret], nil
}

func (e *Encoder) SetBitrate(bitrate int) error {
	ret := C.opus_encoder_set_bitrate(e.encoder, C.opus_int32(bitrate))
	if ret != C.OPUS_OK {
		return errors.New("failed to set bitrate")
	}
	return nil
}

func (e *Encoder) SetComplexity(complexity int) error {
	ret := C.opus_encoder_set_complexity(e.encoder, C.int(complexity))
	if ret != C.OPUS_OK {
		return errors.New("failed to set complexity")
	}
	return nil
}

func (e *Encoder) SetVBR(vbr bool) error {
	vbrInt := 0
	if vbr {
		vbrInt = 1
	}
	ret := C.opus_encoder_set_vbr(e.encoder, C.int(vbrInt))
	if ret != C.OPUS_OK {
		return errors.New("failed to set vbr")
	}
	return nil
}

func (e *Encoder) Destroy() {
	if e.encoder != nil {
		C.opus_encoder_destroy(e.encoder)
		e.encoder = nil
	}
}
