package javdb

// Movie holds basic movie info.
type Movie struct {
	Code        string   `json:"code"`
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Rating      float64  `json:"rating"`
	RatingCount int      `json:"rating_count"`
	Date        string   `json:"date"`
	HasMagnet   bool     `json:"has_magnet"`
	ImageURL    string   `json:"image_url"`
	Tags        []string `json:"tags"`
	ReviewID    int      `json:"review_id"`
}

// Actor holds actor info.
type Actor struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	ImageURL string `json:"image_url"`
}

// SearchResult is the search response.
type SearchResult struct {
	Query   string  `json:"query"`
	Movies  []Movie `json:"movies"`
	Total   int     `json:"total"`
	Message string  `json:"message,omitempty"`
}

// RankingResult is the ranking response.
type RankingResult struct {
	Period  string  `json:"period"`
	Type    string  `json:"type"`
	Movies  []Movie `json:"movies"`
	Message string  `json:"message,omitempty"`
}

// BookmarkResult is the bookmark list response.
type BookmarkResult struct {
	Type     string  `json:"type"`
	Movies   []Movie `json:"movies"`
	Actors   []Actor `json:"actors"`
	Total    int     `json:"total"`
	Page     int     `json:"page"`
	HasNext  bool    `json:"has_next"`
	NextPage int     `json:"next_page"`
	Message  string  `json:"message,omitempty"`
}

// MagnetLink holds a magnet link and metadata.
type MagnetLink struct {
	Name    string `json:"name"`
	Size    string `json:"size"`
	Date    string `json:"date"`
	Magnet  string `json:"magnet"`
	IsHD    bool   `json:"is_hd"`
	HasSubs bool   `json:"has_subs"`
}

// Review is a single review entry.
type Review struct {
	ID      int    `json:"id"`
	Author  string `json:"author"`
	Rating  int    `json:"rating"`
	Likes   int    `json:"likes"`
	Content string `json:"content"`
	Date    string `json:"date"`
}

// ReviewResult is the paginated review list response.
type ReviewResult struct {
	Code     string   `json:"code"`
	Page     int      `json:"page"`
	HasPrev  bool     `json:"has_prev"`
	HasNext  bool     `json:"has_next"`
	PrevPage int      `json:"prev_page"`
	NextPage int      `json:"next_page"`
	Reviews  []Review `json:"reviews"`
}

// MovieDetail is the full movie detail including magnets.
type MovieDetail struct {
	Code        string       `json:"code"`
	Title       string       `json:"title"`
	URL         string       `json:"url"`
	Rating      float64      `json:"rating"`
	RatingCount int          `json:"rating_count"`
	Date        string       `json:"date"`
	Duration    string       `json:"duration"`
	Director    string       `json:"director"`
	Publisher   string       `json:"publisher"`
	Series      string       `json:"series"`
	ImageURL    string       `json:"image_url"`
	Tags        []string     `json:"tags"`
	Actors      []Actor      `json:"actors"`
	Magnets     []MagnetLink `json:"magnets"`
	Description string       `json:"description"`
}
