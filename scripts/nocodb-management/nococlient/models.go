package nococlient

type BaseList struct {
	List     []Base `json:"list"`
	PageInfo struct {
		IsFirstPage bool `json:"isFirstPage"`
		IsLastPage  bool `json:"isLastPage"`
		Page        int  `json:"page"`
		PageSize    int  `json:"pageSize"`
		TotalRows   int  `json:"totalRows"`
	} `json:"pageInfo"`
}

type Source struct {
	Alias            string `json:"alias"`
	Config           any    `json:"config"`
	CreatedAt        string `json:"created_at"`
	Enabled          int    `json:"enabled"`
	ID               string `json:"id"`
	InflectionColumn string `json:"inflection_column"`
	InflectionTable  string `json:"inflection_table"`
	// IsMeta           bool   `json:"is_meta"`
	IsMeta    int    `json:"is_meta"`
	Order     int    `json:"order"`
	BaseID    string `json:"base_id"`
	Type      string `json:"type"`
	UpdatedAt string `json:"updated_at"`
}

type Base struct {
	Sources     []Source `json:"sources"`
	Color       string   `json:"color"`
	CreatedAt   string   `json:"created_at"`
	Deleted     int      `json:"deleted"`
	Description string   `json:"description"`
	ID          string   `json:"id"`
	IsMeta      int      `json:"is_meta"`
	// Meta        struct {
	// } `json:"meta"`
	Order     int    `json:"order"`
	Prefix    string `json:"prefix"`
	Status    string `json:"status"`
	Title     string `json:"title"`
	UpdatedAt string `json:"updated_at"`
}
