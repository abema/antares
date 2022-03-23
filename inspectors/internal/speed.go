package internal

type TimePoint struct {
	RealTime  float64
	VideoTime float64
	SegmentID interface{}
}

type Speedometer struct {
	timePoints []*TimePoint
	interval   float64
}

func NewSpeedometer(interval float64) *Speedometer {
	return &Speedometer{
		timePoints: make([]*TimePoint, 0, 8),
		interval:   interval,
	}
}

func (m *Speedometer) LatestTimePoint() *TimePoint {
	if len(m.timePoints) == 0 {
		return nil
	}
	return m.timePoints[len(m.timePoints)-1]
}

func (m *Speedometer) AddTimePoint(tp *TimePoint) {
	m.timePoints = append(m.timePoints, tp)
	for i := 1; i < len(m.timePoints); i++ {
		if m.timePoints[i].RealTime > tp.RealTime-m.interval {
			m.timePoints = m.timePoints[i-1:]
			break
		}
	}
}

func (m *Speedometer) Satisfied() bool {
	return len(m.timePoints) >= 2
}

func (m *Speedometer) Gap() float64 {
	return m.VideoTimeElapsed() - m.RealTimeElapsed()
}

func (m *Speedometer) RealTimeElapsed() float64 {
	if len(m.timePoints) < 2 {
		return 0
	}
	old := m.timePoints[0]
	curr := m.timePoints[len(m.timePoints)-1]
	return curr.RealTime - old.RealTime
}

func (m *Speedometer) VideoTimeElapsed() float64 {
	if len(m.timePoints) < 2 {
		return 0
	}
	old := m.timePoints[0]
	curr := m.timePoints[len(m.timePoints)-1]
	return curr.VideoTime - old.VideoTime
}
