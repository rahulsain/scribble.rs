package frontend

import (
	"log"
	"net/http"

	"github.com/scribble-rs/scribble.rs/internal/api"
	"github.com/scribble-rs/scribble.rs/internal/game"
	"github.com/scribble-rs/scribble.rs/internal/state"
	"github.com/scribble-rs/scribble.rs/internal/translations"
)

// This file contains the API for the official web client.

// homePage servers the default page for scribble.rs, which is the page to
// create a new lobby.
func homePage(writer http.ResponseWriter, request *http.Request) {
	translation, locale := determineTranslation(request)
	createPageData := createDefaultLobbyCreatePageData()
	createPageData.Translation = translation
	createPageData.Locale = locale

	err := pageTemplates.ExecuteTemplate(writer, "lobby-create-page", createPageData)
	if err != nil {
		log.Printf("Error templating home page: %s\n", err)
	}
}

func createDefaultLobbyCreatePageData() *LobbyCreatePageData {
	return &LobbyCreatePageData{
		BasePageConfig:    currentBasePageConfig,
		SettingBounds:     game.LobbySettingBounds,
		Languages:         game.SupportedLanguages,
		Public:            "false",
		DrawingTime:       "120",
		Rounds:            "4",
		MaxPlayers:        "12",
		CustomWordsChance: "50",
		ClientsPerIPLimit: "1",
		EnableVotekick:    "true",
		Language:          "english",
	}
}

// LobbyCreatePageData defines all non-static data for the lobby create page.
type LobbyCreatePageData struct {
	*BasePageConfig
	*game.SettingBounds
	Translation       translations.Translation
	Locale            string
	Errors            []string
	Languages         map[string]string
	Public            string
	DrawingTime       string
	Rounds            string
	MaxPlayers        string
	CustomWords       string
	CustomWordsChance string
	ClientsPerIPLimit string
	EnableVotekick    string
	Language          string
}

// ssrCreateLobby allows creating a lobby, optionally returning errors that
// occurred during creation.
func ssrCreateLobby(writer http.ResponseWriter, request *http.Request) {
	formParseError := request.ParseForm()
	if formParseError != nil {
		http.Error(writer, formParseError.Error(), http.StatusBadRequest)
		return
	}

	language, languageInvalid := api.ParseLanguage(request.Form.Get("language"))
	drawingTime, drawingTimeInvalid := api.ParseDrawingTime(request.Form.Get("drawing_time"))
	rounds, roundsInvalid := api.ParseRounds(request.Form.Get("rounds"))
	maxPlayers, maxPlayersInvalid := api.ParseMaxPlayers(request.Form.Get("max_players"))
	customWords, customWordsInvalid := api.ParseCustomWords(request.Form.Get("custom_words"))
	customWordChance, customWordChanceInvalid := api.ParseCustomWordsChance(request.Form.Get("custom_words_chance"))
	clientsPerIPLimit, clientsPerIPLimitInvalid := api.ParseClientsPerIPLimit(request.Form.Get("clients_per_ip_limit"))
	enableVotekick, enableVotekickInvalid := api.ParseBoolean("enable votekick", request.Form.Get("enable_votekick"))
	publicLobby, publicLobbyInvalid := api.ParseBoolean("public", request.Form.Get("public"))

	// Prevent resetting the form, since that would be annoying as hell.
	pageData := LobbyCreatePageData{
		BasePageConfig:    currentBasePageConfig,
		SettingBounds:     game.LobbySettingBounds,
		Languages:         game.SupportedLanguages,
		Public:            request.Form.Get("public"),
		DrawingTime:       request.Form.Get("drawing_time"),
		Rounds:            request.Form.Get("rounds"),
		MaxPlayers:        request.Form.Get("max_players"),
		CustomWords:       request.Form.Get("custom_words"),
		CustomWordsChance: request.Form.Get("custom_words_chance"),
		ClientsPerIPLimit: request.Form.Get("clients_per_ip_limit"),
		EnableVotekick:    request.Form.Get("enable_votekick"),
		Language:          request.Form.Get("language"),
	}

	if languageInvalid != nil {
		pageData.Errors = append(pageData.Errors, languageInvalid.Error())
	}
	if drawingTimeInvalid != nil {
		pageData.Errors = append(pageData.Errors, drawingTimeInvalid.Error())
	}
	if roundsInvalid != nil {
		pageData.Errors = append(pageData.Errors, roundsInvalid.Error())
	}
	if maxPlayersInvalid != nil {
		pageData.Errors = append(pageData.Errors, maxPlayersInvalid.Error())
	}
	if customWordsInvalid != nil {
		pageData.Errors = append(pageData.Errors, customWordsInvalid.Error())
	}
	if customWordChanceInvalid != nil {
		pageData.Errors = append(pageData.Errors, customWordChanceInvalid.Error())
	}
	if clientsPerIPLimitInvalid != nil {
		pageData.Errors = append(pageData.Errors, clientsPerIPLimitInvalid.Error())
	}
	if enableVotekickInvalid != nil {
		pageData.Errors = append(pageData.Errors, enableVotekickInvalid.Error())
	}
	if publicLobbyInvalid != nil {
		pageData.Errors = append(pageData.Errors, publicLobbyInvalid.Error())
	}

	translation, locale := determineTranslation(request)
	pageData.Translation = translation
	pageData.Locale = locale

	if len(pageData.Errors) != 0 {
		err := pageTemplates.ExecuteTemplate(writer, "lobby-create-page", pageData)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	playerName := api.GetPlayername(request)

	player, lobby, createError := game.CreateLobby(playerName, language, publicLobby, drawingTime, rounds, maxPlayers, customWordChance, clientsPerIPLimit, customWords, enableVotekick)
	if createError != nil {
		pageData.Errors = append(pageData.Errors, createError.Error())
		templateError := pageTemplates.ExecuteTemplate(writer, "lobby-create-page", pageData)
		if templateError != nil {
			userFacingError(writer, templateError.Error())
		}

		return
	}

	lobby.WriteJSON = api.WriteJSON
	player.SetLastKnownAddress(api.GetIPAddressFromRequest(request))

	api.SetUsersessionCookie(writer, player)

	// We only add the lobby if we could do all necessary pre-steps successfully.
	state.AddLobby(lobby)

	http.Redirect(writer, request, currentBasePageConfig.RootPath+"/ssrEnterLobby?lobby_id="+lobby.LobbyID, http.StatusFound)
}
