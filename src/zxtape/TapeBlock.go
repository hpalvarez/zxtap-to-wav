package zxtape

import (
	"bytes"
	"io"
	wav "github.com/raydac/zxtap-wav"
	zx "github.com/raydac/zxtap-zx"
)

type TapeBlock struct {
	Data     *[]byte
	Checksum byte
}

func writeDataByte(data byte, hi byte, lo byte, writer *bytes.Buffer, freq int, tk90turbo bool) error {
	
	var pulselen_zero int = 855
	var pulselen_one int = 1710	
	
	if tk90turbo {
		pulselen_zero = 325
		pulselen_one = 649
	}
	

	var mask byte = 0x80
	for mask != 0 {
		var len int
		if (data & mask) == 0 {
			len = pulselen_zero
		} else {
			len = pulselen_one
		}

		if err := wav.DoSignal(writer, hi, len, freq); err != nil {
			return err
		}
		if err := wav.DoSignal(writer, lo, len, freq); err != nil {
			return err
		}
		mask >>= 1
	}
	return nil
}

func (t *TapeBlock) SaveSoundData(amplify bool, soundBuffer *bytes.Buffer, freq int, tk90turbo bool) error {
	
	var pulselen_pilot int = 2168
	var pulselen_sync1 int = 667
	var pulselen_sync2 int = 735
	var pulselen_sync3 int = 954
	var impulsenumber_pilot_header int = 8063
	var impulsenumber_pilot_data int = 3223
	
	if tk90turbo {
		pulselen_pilot = 1408
		pulselen_sync1 = 397
		pulselen_sync2 = 317
		pulselen_sync3 = 954
		impulsenumber_pilot_header = 4835
		impulsenumber_pilot_data = 4835		
	}

	var err error

	var pilotImpulses int
	if (*t.Data)[0] < 128 {
		pilotImpulses = impulsenumber_pilot_header
	} else {
		pilotImpulses = impulsenumber_pilot_data
	}

	var HI, LO byte
	if amplify {
		HI = 0xFF
		LO = 0x00
	} else {
		HI = 0xC0
		LO = 0x40
	}

	var signalState = HI

	for i := 0; i < pilotImpulses; i++ {
		if err = wav.DoSignal(soundBuffer, signalState, pulselen_pilot, freq); err != nil {
			return err
		}

		if signalState == HI {
			signalState = LO
		} else {
			signalState = HI
		}
	}

	if signalState == LO {
		if err = wav.DoSignal(soundBuffer, LO, pulselen_pilot, freq); err != nil {
			return err
		}
	}

	if err = wav.DoSignal(soundBuffer, HI, pulselen_sync1, freq); err != nil {
		return err
	}

	if err = wav.DoSignal(soundBuffer, LO, pulselen_sync2, freq); err != nil {
		return err
	}

	for _, d := range *t.Data {
		if err = writeDataByte(d, HI, LO, soundBuffer, freq, tk90turbo); err != nil {
			return err
		}
	}

	if err = writeDataByte(t.Checksum, HI, LO, soundBuffer, freq, tk90turbo); err != nil {
		return err
	}

	if err = wav.DoSignal(soundBuffer, HI, pulselen_sync3, freq); err != nil {
		return err
	}

	return nil
}

func ReadTapeBlock(reader io.Reader) (*TapeBlock, error) {
	var length int
	var err error
	var checksum byte

	length, err = zx.ReadZxShort(reader)
	if err != nil {
		return nil, err
	}

	data := make([]byte, length-1)

	_, err = io.ReadAtLeast(reader, data, len(data))
	if err != nil {
		return nil, err
	}

	checksum, err = zx.ReadZxByte(reader)
	if err != nil {
		return nil, err
	}

	return &TapeBlock{&data, checksum}, nil
}
