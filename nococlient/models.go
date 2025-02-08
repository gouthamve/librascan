package nococlient

import "time"

type BaseList struct {
	List     []Base   `json:"list"`
	PageInfo PageInfo `json:"pageInfo"`
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

type TableList struct {
	List []struct {
		ID        string    `json:"id"`
		SourceID  string    `json:"source_id"`
		BaseID    string    `json:"base_id"`
		TableName string    `json:"table_name"`
		Title     string    `json:"title"`
		Type      string    `json:"type"`
		Meta      any       `json:"meta"`
		Schema    any       `json:"schema"`
		Enabled   bool      `json:"enabled"`
		Mm        bool      `json:"mm"`
		Tags      any       `json:"tags"`
		Pinned    any       `json:"pinned"`
		Deleted   any       `json:"deleted"`
		Order     int       `json:"order"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	} `json:"list"`
	PageInfo PageInfo `json:"pageInfo"`
}

type TableCreateRequest struct {
	Columns   []Column `json:"columns"`
	TableName string   `json:"table_name"`
	Title     string   `json:"title"`
}

type Column struct {
	Ai         bool   `json:"ai"`
	Altered    int    `json:"altered"`
	Cdf        string `json:"cdf"`
	Ck         bool   `json:"ck"`
	Clen       int    `json:"clen"`
	ColumnName string `json:"column_name"`
	Ct         string `json:"ct"`
	Dt         string `json:"dt"`
	Dtx        string `json:"dtx"`
	Dtxp       string `json:"dtxp"`
	Dtxs       string `json:"dtxs"`
	Np         any    `json:"np"`
	Nrqd       bool   `json:"nrqd"`
	Ns         any    `json:"ns"`
	Pk         bool   `json:"pk"`
	Rqd        bool   `json:"rqd"`
	Title      string `json:"title"`
	Uicn       string `json:"uicn"`
	Uidt       string `json:"uidt"`
	Uip        string `json:"uip"`
	Un         bool   `json:"un"`
}

type PageInfo struct {
	IsFirstPage bool `json:"isFirstPage"`
	IsLastPage  bool `json:"isLastPage"`
	Page        int  `json:"page"`
	PageSize    int  `json:"pageSize"`
	TotalRows   int  `json:"totalRows"`
}

type Book struct {
	ISBN          string   `json:"id"`
	Title         string   `json:"title"`
	Author        []string `json:"author"`
	Description   string   `json:"description"`
	PublishedDate string   `json:"published_date"`
	Category      []string `json:"category"`
}
