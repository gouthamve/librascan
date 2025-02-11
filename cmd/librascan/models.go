package main

type GoogleBooksResponse struct {
	Kind       string       `json:"kind"`
	TotalItems int          `json:"totalItems"`
	Items      []GoogleBook `json:"items"`
}

type GoogleBook struct {
	Kind       string `json:"kind"`
	ID         string `json:"id"`
	Etag       string `json:"etag"`
	SelfLink   string `json:"selfLink"`
	VolumeInfo struct {
		Title               string   `json:"title"`
		Subtitle            string   `json:"subtitle"`
		Authors             []string `json:"authors"`
		Publisher           string   `json:"publisher"`
		PublishedDate       string   `json:"publishedDate"`
		Description         string   `json:"description"`
		IndustryIdentifiers []struct {
			Type       string `json:"type"`
			Identifier string `json:"identifier"`
		} `json:"industryIdentifiers"`
		ReadingModes struct {
			Text  bool `json:"text"`
			Image bool `json:"image"`
		} `json:"readingModes"`
		PageCount           int      `json:"pageCount"`
		PrintType           string   `json:"printType"`
		Categories          []string `json:"categories"`
		MaturityRating      string   `json:"maturityRating"`
		AllowAnonLogging    bool     `json:"allowAnonLogging"`
		ContentVersion      string   `json:"contentVersion"`
		PanelizationSummary struct {
			ContainsEpubBubbles  bool `json:"containsEpubBubbles"`
			ContainsImageBubbles bool `json:"containsImageBubbles"`
		} `json:"panelizationSummary"`
		ImageLinks struct {
			SmallThumbnail string `json:"smallThumbnail"`
			Thumbnail      string `json:"thumbnail"`
		} `json:"imageLinks"`
		Language            string `json:"language"`
		PreviewLink         string `json:"previewLink"`
		InfoLink            string `json:"infoLink"`
		CanonicalVolumeLink string `json:"canonicalVolumeLink"`
	} `json:"volumeInfo"`
	SaleInfo struct {
		Country     string `json:"country"`
		Saleability string `json:"saleability"`
		IsEbook     bool   `json:"isEbook"`
	} `json:"saleInfo"`
	AccessInfo struct {
		Country                string `json:"country"`
		Viewability            string `json:"viewability"`
		Embeddable             bool   `json:"embeddable"`
		PublicDomain           bool   `json:"publicDomain"`
		TextToSpeechPermission string `json:"textToSpeechPermission"`
		Epub                   struct {
			IsAvailable bool `json:"isAvailable"`
		} `json:"epub"`
		Pdf struct {
			IsAvailable bool `json:"isAvailable"`
		} `json:"pdf"`
		WebReaderLink       string `json:"webReaderLink"`
		AccessViewStatus    string `json:"accessViewStatus"`
		QuoteSharingAllowed bool   `json:"quoteSharingAllowed"`
	} `json:"accessInfo"`
	SearchInfo struct {
		TextSnippet string `json:"textSnippet"`
	} `json:"searchInfo"`
}

type OpenLibraryResponse map[string]OpenLibraryBook

type OpenLibraryBook struct {
	URL      string `json:"url"`
	Key      string `json:"key"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Authors  []struct {
		URL  string `json:"url"`
		Name string `json:"name"`
	} `json:"authors"`
	Pagination  string `json:"pagination"`
	Weight      string `json:"weight"`
	Identifiers struct {
		Isbn13      []string `json:"isbn_13"`
		Openlibrary []string `json:"openlibrary"`
	} `json:"identifiers"`
	Publishers []struct {
		Name string `json:"name"`
	} `json:"publishers"`
	PublishDate string `json:"publish_date"`
	Subjects    []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"subjects"`
	Cover struct {
		Small  string `json:"small"`
		Medium string `json:"medium"`
		Large  string `json:"large"`
	} `json:"cover"`
}

type DebugResponse struct {
	Book                Book                 `json:"book"`
	GoogleBooksResponse *GoogleBooksResponse `json:"google_books_response"`
	OpenLibraryResponse *OpenLibraryResponse `json:"open_library_response"`
}

type Book struct {
	ISBN          string   `json:"isbn"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Author        []string `json:"author"`
	Publisher     string   `json:"publisher"`
	PublishedDate string   `json:"published_date"`
	Categories    []string `json:"categories"`
	Pages         int      `json:"pages"`
	Language      string   `json:"language"`
	CoverURL      string   `json:"cover"`
}

type Shelf struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	RowCount int    `json:"rows_count"`
}
