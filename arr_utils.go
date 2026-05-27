package jet

const (
	ErrCodeSplitLengthExceeded = "ARR-001"
)

var (
	ErrSplitLengthExceeded = func() error {
		return NewAppErrBuilder(ErrCodeSplitLengthExceeded, "split length exceeded").Business().Err()
	}
)

func SplitArr[T any](data []T, size int) [][]T {
	if len(data) == 0 {
		return nil
	}
	var chunks [][]T
	for i := 0; i < len(data); i += size {
		end := i + size
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}
	return chunks
}

func SplitArrByItemLen[T ~string](data []T, max int) ([][]T, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var chunks [][]T
	var curArr []T
	curLen := 0
	for i := 0; i < len(data); i += 1 {
		if len(data[i]) > max {
			return nil, ErrSplitLengthExceeded()
		}
		if curLen+len(data[i]) <= max {
			curLen += len(data[i])
			curArr = append(curArr, data[i])
		} else {
			chunks = append(chunks, curArr)
			curLen = len(data[i])
			curArr = []T{data[i]}
		}
	}
	if len(curArr) != 0 {
		chunks = append(chunks, curArr)
	}
	return chunks, nil
}
