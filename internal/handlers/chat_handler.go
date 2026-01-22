package handlers

import (
	"encoding/json"
	"errors"
	"naimuBack/internal/models"
	service "naimuBack/internal/services"
	"net/http"
	"strconv"
)

type ChatHandler struct {
	ChatService *service.ChatService
}

func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	var chat models.Chat
	err := json.NewDecoder(r.Body).Decode(&chat)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	createdChat, err := h.ChatService.CreateChat(r.Context(), chat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdChat)
}

func (h *ChatHandler) GetChatByID(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get(":id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	chat, err := h.ChatService.GetChatByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrCategoryNotFound) {
			http.Error(w, "Chat not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to retrieve chat", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chat)
}

func (h *ChatHandler) GetAllChats(w http.ResponseWriter, r *http.Request) {
	chats, err := h.ChatService.GetAllChats(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve chats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chats)
}

func (h *ChatHandler) GetChatsByUserID(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(idParam)
	if err != nil || userID <= 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	requesterID, _ := r.Context().Value("user_id").(int)
	requesterRole, _ := r.Context().Value("role").(string)
	if requesterRole == "business_worker" && requesterID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	chats, err := h.ChatService.GetChatsByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to retrieve chats", http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, 0, len(chats))

	for _, chat := range chats {
		response = append(response, buildChatResponseItem(chat))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"chats": response})
}

func buildChatResponseItem(chat models.AdChats) map[string]interface{} {
	item := map[string]interface{}{
		"ad_name":      chat.AdName,
		"status":       chat.Status,
		"hide_phone":   chat.HidePhone,
		"is_author":    chat.IsAuthor,
		"performer_id": chat.PerformerID,
		"users":        chat.Users,
		"ad_type":      chat.AdType,
	}

	addStringIfNotEmpty(item, "address", chat.Address)
	addStringIfNotEmpty(item, "city_name", chat.CityName)

	if len(chat.Images) > 0 {
		item["images"] = chat.Images
	}
	if len(chat.Videos) > 0 {
		item["videos"] = chat.Videos
	}
	if chat.CreatedAt != nil {
		item["created_at"] = chat.CreatedAt
	}

	if chat.AdID != nil {
		item["ad_id"] = *chat.AdID
	}
	if chat.ServiceID != nil {
		item["service_id"] = *chat.ServiceID
	}
	if chat.RentAdID != nil {
		item["rentad_id"] = *chat.RentAdID
	}
	if chat.WorkAdID != nil {
		item["workad_id"] = *chat.WorkAdID
	}
	if chat.RentID != nil {
		item["rent_id"] = *chat.RentID
	}
	if chat.WorkID != nil {
		item["work_id"] = *chat.WorkID
	}

	switch chat.AdType {
	case "service":
		item["service_name"] = chat.AdName
		addStringIfNotEmpty(item, "service_address", chat.Address)
		addFloatIfPresent(item, "service_price", chat.Price)
		addFloatIfPresent(item, "price_to", chat.PriceTo)
		item["negotiable"] = chat.Negotiable
		item["on_site"] = chat.OnSite
		addStringIfNotEmpty(item, "service_description", chat.Description)
		addStringIfNotEmpty(item, "work_time_from", chat.WorkTimeFrom)
		addStringIfNotEmpty(item, "work_time_to", chat.WorkTimeTo)
		addStringPointer(item, "latitude", chat.Latitude)
		addStringPointer(item, "longitude", chat.Longitude)
	case "rent_ad":
		item["rentad_name"] = chat.AdName
		addStringIfNotEmpty(item, "rentad_address", chat.Address)
		addFloatIfPresent(item, "rentad_price", chat.Price)
		addFloatIfPresent(item, "price_to", chat.PriceTo)
		item["negotiable"] = chat.Negotiable
		addStringIfNotEmpty(item, "rentad_description", chat.Description)
		addStringIfNotEmpty(item, "work_time_from", chat.WorkTimeFrom)
		addStringIfNotEmpty(item, "work_time_to", chat.WorkTimeTo)
		addStringIfNotEmpty(item, "rent_type", chat.RentType)
		addStringIfNotEmpty(item, "deposit", chat.Deposit)
		addStringPointer(item, "latitude", chat.Latitude)
		addStringPointer(item, "longitude", chat.Longitude)
	case "work_ad":
		item["workad_name"] = chat.AdName
		addStringIfNotEmpty(item, "workad_address", chat.Address)
		addFloatIfPresent(item, "workad_price", chat.Price)
		addFloatIfPresent(item, "workad_price_to", chat.PriceTo)
		item["negotiable"] = chat.Negotiable
		addStringIfNotEmpty(item, "workad_description", chat.Description)
		addStringIfNotEmpty(item, "work_experience", chat.WorkExperience)
		addStringIfNotEmpty(item, "schedule", chat.Schedule)
		addStringIfNotEmpty(item, "distance_work", chat.DistanceWork)
		addStringIfNotEmpty(item, "payment_period", chat.PaymentPeriod)
		addStringIfNotEmpty(item, "work_time_from", chat.WorkTimeFrom)
		addStringIfNotEmpty(item, "work_time_to", chat.WorkTimeTo)
		addStringPointer(item, "latitude", chat.Latitude)
		addStringPointer(item, "longitude", chat.Longitude)
	case "rent":
		item["rent_name"] = chat.AdName
		addStringIfNotEmpty(item, "rent_address", chat.Address)
		addFloatIfPresent(item, "rent_price", chat.Price)
		addFloatIfPresent(item, "rent_price_to", chat.PriceTo)
		item["negotiable"] = chat.Negotiable
		addStringIfNotEmpty(item, "rent_description", chat.Description)
		addStringIfNotEmpty(item, "work_time_from", chat.WorkTimeFrom)
		addStringIfNotEmpty(item, "work_time_to", chat.WorkTimeTo)
		addStringIfNotEmpty(item, "rent_type", chat.RentType)
		addStringIfNotEmpty(item, "deposit", chat.Deposit)
		addStringPointer(item, "latitude", chat.Latitude)
		addStringPointer(item, "longitude", chat.Longitude)
	case "work":
		item["work_name"] = chat.AdName
		addStringIfNotEmpty(item, "work_address", chat.Address)
		addFloatIfPresent(item, "work_price", chat.Price)
		addFloatIfPresent(item, "work_price_to", chat.PriceTo)
		item["negotiable"] = chat.Negotiable
		addStringIfNotEmpty(item, "work_description", chat.Description)
		addStringIfNotEmpty(item, "work_experience", chat.WorkExperience)
		addStringIfNotEmpty(item, "schedule", chat.Schedule)
		addStringIfNotEmpty(item, "distance_work", chat.DistanceWork)
		addStringIfNotEmpty(item, "payment_period", chat.PaymentPeriod)
		addStringIfNotEmpty(item, "work_time_from", chat.WorkTimeFrom)
		addStringIfNotEmpty(item, "work_time_to", chat.WorkTimeTo)
		addStringPointer(item, "latitude", chat.Latitude)
		addStringPointer(item, "longitude", chat.Longitude)
	default:
		item["ad_name"] = chat.AdName
		addStringIfNotEmpty(item, "ad_address", chat.Address)
		addFloatIfPresent(item, "ad_price", chat.Price)
		addFloatIfPresent(item, "price_to", chat.PriceTo)
		item["negotiable"] = chat.Negotiable
		item["on_site"] = chat.OnSite
		addStringIfNotEmpty(item, "ad_description", chat.Description)
		addStringIfNotEmpty(item, "work_time_from", chat.WorkTimeFrom)
		addStringIfNotEmpty(item, "work_time_to", chat.WorkTimeTo)
		addStringPointer(item, "latitude", chat.Latitude)
		addStringPointer(item, "longitude", chat.Longitude)
	}

	return item
}

func addStringIfNotEmpty(m map[string]interface{}, key, value string) {
	if value != "" {
		m[key] = value
	}
}

func addStringPointer(m map[string]interface{}, key string, value *string) {
	if value != nil && *value != "" {
		m[key] = *value
	}
}

func addFloatIfPresent(m map[string]interface{}, key string, value *float64) {
	if value != nil {
		m[key] = *value
	}
}

func (h *ChatHandler) DeleteChat(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get(":id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	err = h.ChatService.DeleteChat(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to delete chat", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetPerformerChats proxies listing chats for business and its workers.
func (h *ChatHandler) GetPerformerChats(w http.ResponseWriter, r *http.Request) {
	businessUserID, _ := r.Context().Value("user_id").(int)
	chats, err := h.ChatService.GetChatsByUserID(r.Context(), businessUserID)
	if err != nil {
		http.Error(w, "Failed to retrieve chats", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"chats": chats})
}

// GetWorkerChats returns base business-worker chat ids.
func (h *ChatHandler) GetWorkerChats(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("user_id").(int)
	role, _ := r.Context().Value("role").(string)

	chats, err := h.ChatService.GetWorkerChats(r.Context(), userID, role)
	if err != nil {
		if err.Error() == "forbidden" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"chats": chats})
}
