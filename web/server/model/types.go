package model

type Content struct {
	Hero        HeroSection    `json:"hero"`
	Services    []Service      `json:"services"`
	Solutions   []Solution     `json:"solutions"`
	ChatService ChatService    `json:"chatService"`
	Trust       TrustSection   `json:"trust"`
	Process     []ProcessStep  `json:"process"`
	About       AboutSection   `json:"about"`
	Contact     ContactSection `json:"contact"`
}

type HeroSection struct {
	Title         string `json:"title"`
	Subtitle      string `json:"subtitle"`
	CtaPrimary    string `json:"ctaPrimary"`
	CtaSecondary  string `json:"ctaSecondary"`
}

type Service struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type Solution struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

type ChatService struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	TechStack   []string `json:"techStack"`
}

type TrustSection struct {
	Metrics []TrustMetric `json:"metrics"`
	Badges  []string      `json:"badges"`
}

type TrustMetric struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type ProcessStep struct {
	Step        int    `json:"step"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type AboutSection struct {
	Mission     string `json:"mission"`
	Vision      string `json:"vision"`
	Description string `json:"description"`
}

type ContactSection struct {
	Email   string `json:"email"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
}

type Inquiry struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type ContactRequest struct {
	Name    string `json:"name" binding:"required"`
	Email   string `json:"email" binding:"required,email"`
	Message string `json:"message" binding:"required"`
}

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
