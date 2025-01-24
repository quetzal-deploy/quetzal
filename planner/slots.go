package planner

import "time"

type Slot struct {
	Chans []chan bool
}

func newSlot() Slot {
	return Slot{
		Chans: make([]chan bool, 0),
	}
}

func (s *Slot) Free() {
	// Read from al all channels to free up the slots
	// IMPORTANT: The specific order of channels being read from can cause deadlocks.
	//            Read from the channels round-robin and keep track of which ones are already done

	// Keep track of which channels have already been read from
	chansOK := make([]bool, len(s.Chans))
	for i, _ := range chansOK {
		chansOK[i] = false
	}

	// Number of channels that have been read from
	okCount := 0

	for okCount < len(s.Chans) {
		for i, c := range s.Chans {
			if !chansOK[i] {
				select {
				case <-c:
					chansOK[i] = true
					okCount += 1
				default:
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
	}
}

func (s *Slot) AddChannel(c chan bool) {
	s.Chans = append(s.Chans, c)
}
