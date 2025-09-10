package handlers

type CrashHistory struct {
	data []float64
	size int
}

func NewCrashHistory(limit int) *CrashHistory {
	return &CrashHistory{
		data: make([]float64, 0, limit),
		size: limit,
	}
}

func (h *CrashHistory) Add(value float64) {
	if len(h.data) >= h.size {
		h.data = h.data[1:]
	}
	h.data = append(h.data, value)
}

func (h *CrashHistory) GetAll() []float64 {
	return h.data
}
