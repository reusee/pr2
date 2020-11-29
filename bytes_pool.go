package pr

func NewBytesPool(size int64, capacity int32) (
	get func() (
		bs []byte,
		put func(),
	),
) {

	getBytes := NewPool(capacity, func() any {
		return make([]byte, size)
	})

	get = func() (bs []byte, put func()) {
		v, put := getBytes()
		return v.([]byte), put
	}

	return
}
