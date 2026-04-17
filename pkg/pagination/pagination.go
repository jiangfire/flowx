package pagination

// PaginatedResult 分页结果
type PaginatedResult struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// NormalizePage 规范化分页参数
func NormalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

// CalcTotalPages 计算总页数
func CalcTotalPages(total int64, pageSize int) int {
	if pageSize < 1 {
		pageSize = 20
	}
	return int((total + int64(pageSize) - 1) / int64(pageSize))
}

// Offset 计算分页偏移量
func Offset(page, pageSize int) int {
	return (page - 1) * pageSize
}

// NewResult 创建分页结果
func NewResult(total int64, page, pageSize int) *PaginatedResult {
	return &PaginatedResult{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: CalcTotalPages(total, pageSize),
	}
}
