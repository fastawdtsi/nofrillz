package api

import (
	"net/http"

	"nofrillz/internal/app"
)

type Topic struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Image       string `json:"image"`
}

type topicsResponse struct {
	Topics []Topic `json:"topics"`
}

type TopicsHandler struct {
	app *app.App
}

func NewTopicsHandler(app *app.App) *TopicsHandler {
	return &TopicsHandler{
		app: app,
	}
}

func (h *TopicsHandler) AddRoutes(sm *http.ServeMux, middleware func(http.HandlerFunc) http.HandlerFunc) {
	sm.HandleFunc("GET /topics", middleware(h.List))
}

func (h *TopicsHandler) List(w http.ResponseWriter, r *http.Request) {
	ip := ClientIP(r)
	if !AllowReadByIP(w, h.app, ip) {
		return
	}

	Respond(w, http.StatusOK, topicsResponse{
		Topics: hardcodedTopics(),
	}, h.app.Logger)
}

func hardcodedTopics() []Topic {
	return []Topic{
		{
			Name:        "Technology",
			Description: "Product launches, startups, gadgets, and the ideas shaping what comes next.",
			Image:       "https://images.nofrillz.dev/topics/technology.jpg",
		},
		{
			Name:        "Design",
			Description: "Brand systems, interiors, visual culture, and the details that make things feel intentional.",
			Image:       "https://images.nofrillz.dev/topics/design.jpg",
		},
		{
			Name:        "Travel",
			Description: "Destinations, neighborhood finds, and perspective-shifting trips worth talking about.",
			Image:       "https://images.nofrillz.dev/topics/travel.jpg",
		},
		{
			Name:        "Food",
			Description: "Recipes, restaurants, kitchen rituals, and the kind of meals people want to remember.",
			Image:       "https://images.nofrillz.dev/topics/food.jpg",
		},
		{
			Name:        "Fitness",
			Description: "Training, recovery, movement, and routines that help people stay consistent.",
			Image:       "https://images.nofrillz.dev/topics/fitness.jpg",
		},
		{
			Name:        "Music",
			Description: "Albums, live sets, playlists, and the artists setting the mood right now.",
			Image:       "https://images.nofrillz.dev/topics/music.jpg",
		},
	}
}
