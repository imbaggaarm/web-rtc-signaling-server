package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

const (
	APITypeLogout            = "LOGOUT"
	APITypeLogin             = "LOGIN"
	APITypeProfile           = "PROFILE"
	APITypeFriends           = "FRIENDS"
	APITypeOffer             = "OFFER"
	APITypeOfferResponse     = "OFFER_RESPONSE"
	APITypeAnswer            = "ANSWER"
	APITypeCandidate         = "CANDIDATE"
	APITypeOnlineStateChange = "ONLINE_STATE_CHANGE"
	APITypeFriendRequest     = "FRIEND_REQUEST"
	APITypeRegister          = "REGISTER"
	APITypeUpdateUserProfile = "UPDATE_PROFILE"

	APIParameterKeyEmail    = "email"
	APIParameterKeyPassword = "password"
	APIParameterKeyUsername = "username"
	APIParameterKeyToken    = "token"

	APIErrorWrongAuthentication  = "Wrong email or password"
	APIErrorUserNotValid         = "Username is not valid"
	APIErrorUserExisted          = "User existed"
	APIErrorAuthenticationFailed = "Authentication failed"

	UserOnlineStateOffline      = 0
	UserOnlineStateOnline       = 1
	UserOnlineStateDoNotDisturb = 2
)

type (
	APIType  = string
	APIError = string

	UserProfile struct {
		Username          string `json:"username"`
		Email             string `json:"email"`
		DisplayName       string `json:"display_name"`
		ProfilePictureUrl string `json:"profile_picture_url"`
		CoverPhotoUrl     string `json:"cover_photo_url"`
		OnlineState       int    `json:"online_state"`
	}
	Response struct {
		Type    APIType     `json:"type"`
		Success bool        `json:"success"`
		Data    interface{} `json:"data"`
		Error   APIError    `json:"error"`
	}
	Message struct {
		Type string `json:"type"`
		Data Data   `json:"data"`
	}
	Data struct {
		FromID string `json:"from_id"`
		ToID   string `json:"to_id"`

		Username    string      `json:"username"`
		Candidate   interface{} `json:"candidate"`
		Offer       interface{} `json:"offer"`
		Answer      interface{} `json:"answer"`
		Success     bool        `json:"success"`
		OnlineState int         `json:"online_state"`
	}
	LoginResponse struct {
		JWToken  string `json:"token"`
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	JWT struct {
		Email    string `json:"email"`
		Username string `json:"username"`
		Exp      int64  `json:"exp"`
	}
)

var (
	// Mock data for users name and password
	userAccounts = make(map[string]string)
	// Mock data for username with email
	usernames = make(map[string]string)
	// All current websocket connections
	userConns = make(map[string]*websocket.Conn)
	// All mock user profiles
	userProfiles = make(map[string]*UserProfile)
	// All mock user's friends
	userFriends = make(map[string][]*UserProfile)
	// Websocket upgrader
	upgrader = websocket.Upgrader{}

	broadcast   = make(chan Message)
	userOnlines = make(chan string)
)

func main() {
	fmt.Println("Signaling server")

	r := mux.NewRouter()

	api := r.PathPrefix("/api/v1").Subrouter()
	// Configure websocket route
	api.HandleFunc("/ws", handleWSConnections)
	// Configure auth register route
	api.HandleFunc("/auth/register", handleRegister).Methods(http.MethodPost)
	// Configure auth login route
	api.HandleFunc("/auth/login", handleLogin).Methods(http.MethodPost)
	// Configure auth logout route
	api.HandleFunc("/auth/logout", handleLogout).Methods(http.MethodGet)
	// Configure user route
	api.HandleFunc("/{username}", handleUserProfile).Methods(http.MethodGet)
	// Configure user's friends route
	api.HandleFunc("/{username}/friends", handleFriends).Methods(http.MethodGet)
	// Configure update profile route
	api.HandleFunc("/{username}/update_profile", handleUpdateUserProfile).Methods(http.MethodPost)
	//create mock data with username and password
	userAccounts["user1@gmail.com"] = "123456"
	userAccounts["user2@gmail.com"] = "123456"
	userAccounts["user3@gmail.com"] = "123456"
	userAccounts["user4@gmail.com"] = "123456"

	user1 := UserProfile{
		Username:          "user1",
		Email:             "user1@gmail.com",
		DisplayName:       "Tài Dương",
		ProfilePictureUrl: "https://scontent.fsgn3-1.fna.fbcdn.net/v/t1.0-9/72770272_937416046627629_8601799044018208768_o.jpg?_nc_cat=107&cachebreaker=sd&_nc_oc=AQnwqH0EO0dQARI-ztmAXlPwc8u2WWLIrPG7sSgZlVxyZPVgRSTxU_zAYy0_cWCb8sY&_nc_ht=scontent.fsgn3-1.fna&oh=ef5692abe03095bb89992c91225c110b&oe=5E17544D",
		CoverPhotoUrl:     "https://scontent.fsgn3-1.fna.fbcdn.net/v/t1.0-9/72770272_937416046627629_8601799044018208768_o.jpg?_nc_cat=107&cachebreaker=sd&_nc_oc=AQnwqH0EO0dQARI-ztmAXlPwc8u2WWLIrPG7sSgZlVxyZPVgRSTxU_zAYy0_cWCb8sY&_nc_ht=scontent.fsgn3-1.fna&oh=ef5692abe03095bb89992c91225c110b&oe=5E17544D",
	}
	user2 := UserProfile{
		Username:          "user2",
		Email:             "user2@gmail.com",
		DisplayName:       "Thức Trần",
		ProfilePictureUrl: "https://scontent.fsgn4-1.fna.fbcdn.net/v/t1.0-9/48364556_334991657092752_8475428367296888832_n.jpg?_nc_cat=103&cachebreaker=sd&_nc_oc=AQkOvex4QNZZunBh1zUcLSqxiZFsLH3KQgKAS1fu_c1DSr-uqXjectxRXuDsnGJNYds&_nc_ht=scontent.fsgn4-1.fna&oh=5eca8adb3876a99dbd2d212392408c3a&oe=5E548584",
		CoverPhotoUrl:     "https://scontent.fsgn4-1.fna.fbcdn.net/v/t1.0-9/48364556_334991657092752_8475428367296888832_n.jpg?_nc_cat=103&cachebreaker=sd&_nc_oc=AQkOvex4QNZZunBh1zUcLSqxiZFsLH3KQgKAS1fu_c1DSr-uqXjectxRXuDsnGJNYds&_nc_ht=scontent.fsgn4-1.fna&oh=5eca8adb3876a99dbd2d212392408c3a&oe=5E548584",
	}
	user3 := UserProfile{
		Username:          "user3",
		Email:             "user3@gmail.com",
		DisplayName:       "Công Linh",
		ProfilePictureUrl: "https://scontent.fsgn3-1.fna.fbcdn.net/v/t1.0-9/73148067_1463965623778887_510412543362072576_o.jpg?_nc_cat=111&cachebreaker=sd&_nc_oc=AQlrb5LwhJUpgTz6FvXLwUeU4hzRobK6stNXwd4r8Nf-TDECznkMnRJQ7iJr2C-N8s0&_nc_ht=scontent.fsgn3-1.fna&oh=3d2d871bc31872116e4d3dc363eb3192&oe=5E54A066",
		CoverPhotoUrl:     "https://scontent.fsgn3-1.fna.fbcdn.net/v/t1.0-9/73148067_1463965623778887_510412543362072576_o.jpg?_nc_cat=111&cachebreaker=sd&_nc_oc=AQlrb5LwhJUpgTz6FvXLwUeU4hzRobK6stNXwd4r8Nf-TDECznkMnRJQ7iJr2C-N8s0&_nc_ht=scontent.fsgn3-1.fna&oh=3d2d871bc31872116e4d3dc363eb3192&oe=5E54A066",
	}
	user4 := UserProfile{
		Username:          "user4",
		Email:             "user4@gmail.com",
		DisplayName:       "Tuấn Trần",
		ProfilePictureUrl: "https://scontent.fsgn3-1.fna.fbcdn.net/v/t1.0-1/19366599_851641698321246_3156420856808843114_n.jpg?_nc_cat=107&cachebreaker=sd&_nc_oc=AQmn6BpDCH_hKfl52ZT3AS50SpRuiOvUc78N_4PlRUc3MlXsj363BSrlX8oEo8pQe00&_nc_ht=scontent.fsgn3-1.fna&oh=54befe7439a29ddab67098312266ff98&oe=5E1F1980",
		CoverPhotoUrl:     "https://scontent.fsgn3-1.fna.fbcdn.net/v/t1.0-1/19366599_851641698321246_3156420856808843114_n.jpg?_nc_cat=107&cachebreaker=sd&_nc_oc=AQmn6BpDCH_hKfl52ZT3AS50SpRuiOvUc78N_4PlRUc3MlXsj363BSrlX8oEo8pQe00&_nc_ht=scontent.fsgn3-1.fna&oh=54befe7439a29ddab67098312266ff98&oe=5E1F1980",
	}

	userProfiles["user1"] = &user1
	userProfiles["user2"] = &user2
	userProfiles["user3"] = &user3
	userProfiles["user4"] = &user4

	usernames["user1@gmail.com"] = "user1"
	usernames["user2@gmail.com"] = "user2"
	usernames["user3@gmail.com"] = "user3"
	usernames["user4@gmail.com"] = "user4"

	userFriends["user1"] = []*UserProfile{&user2, &user3, &user4}
	userFriends["user2"] = []*UserProfile{&user1, &user3, &user4}
	userFriends["user3"] = []*UserProfile{&user1, &user2, &user4}
	userFriends["user4"] = []*UserProfile{&user1, &user2, &user3}

	// Handle broadcast message
	go broadcastMessage()
	go setOnlineUser()
	// Start the server on localhost port 8000 and log any error
	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", r)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func writeResponse(w http.ResponseWriter, response Response) {
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(js)
}

func handleUserProfile(w http.ResponseWriter, r *http.Request) {
	pathParams := mux.Vars(r)
	username := ""
	w.Header().Set("Content-Type", "application/json")
	if val, ok := pathParams["username"]; ok {
		username = val
		if user, ok := userProfiles[username]; ok {
			response := Response{
				Type:    APITypeProfile,
				Success: true,
				Data:    user,
				Error:   "",
			}
			writeResponse(w, response)
		} else {
			response := Response{
				Type:    APITypeProfile,
				Success: false,
				Data:    nil,
				Error:   "Username is not valid",
			}
			writeResponse(w, response)
		}
	}
}

func handleUpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	//check auth
	pathParams := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")
	if err := r.ParseForm(); err != nil {
		_, _ = fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}

	jwt := r.FormValue("token")
	jwtObj, err := decodeJWT(jwt)
	if err != nil {
		response := Response{
			Type:    APITypeUpdateUserProfile,
			Success: false,
			Data:    nil,
			Error:   APIErrorAuthenticationFailed,
		}
		writeResponse(w, response)
		return
	}

	if username, ok := pathParams["username"]; ok {
		//get params
		displayName := r.FormValue("display_name")
		profilePictureUrl := r.FormValue("profile_picture_url")
		coverPhotoUrl := r.FormValue("cover_photo_url")
		var uProfile *UserProfile
		if userProfile, ok := userProfiles[username]; ok {
			// update
			uProfile = userProfile
			userProfile.DisplayName = displayName
			userProfile.ProfilePictureUrl = profilePictureUrl
			userProfile.CoverPhotoUrl = coverPhotoUrl

		} else {
			//create new
			uProfile = &UserProfile{
				Username:          username,
				Email:             jwtObj.Email,
				DisplayName:       displayName,
				ProfilePictureUrl: profilePictureUrl,
				CoverPhotoUrl:     coverPhotoUrl,
				OnlineState:       0,
			}

			// set user profile
			userProfiles[username] = uProfile
		}

		// write response
		response := Response{
			Type:    APITypeUpdateUserProfile,
			Success: true,
			Data:    uProfile,
			Error:   "",
		}
		writeResponse(w, response)
	}
}

func handleFriends(w http.ResponseWriter, r *http.Request) {
	pathParams := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")

	username := ""

	if val, ok := pathParams["username"]; ok {
		username = val
		if friends, ok := userFriends[username]; ok {
			response := Response{
				Type:    APITypeFriends,
				Success: true,
				Data:    friends,
				Error:   "",
			}
			writeResponse(w, response)
		} else {
			response := Response{
				Type:    APITypeFriends,
				Success: true,
				Data:    []*UserProfile{},
				Error:   APIErrorUserNotValid,
			}
			writeResponse(w, response)
		}
	}
}

func authenticateUser(email string, password string) (success bool, data *LoginResponse, err string) {
	email = strings.ToLower(email)
	if email == "" {
		err = APIErrorWrongAuthentication
		data = nil
		return
	}
	if pass, ok := userAccounts[email]; ok {
		if password == pass {
			success = true
			exp := time.Now().Unix() + 30*60 //expired after 30 minutes
			var oJWT = JWT{
				Email:    email,
				Username: usernames[email],
				Exp:      exp,
			}
			b, _ := json.Marshal(oJWT)
			jwt := b64.StdEncoding.EncodeToString([]byte(b))
			data = &LoginResponse{
				JWToken:  jwt,
				Username: usernames[email],
				Email:    email,
			}
		} else {
			err = APIErrorWrongAuthentication
			data = nil
		}
	} else {
		err = APIErrorWrongAuthentication
		data = nil
	}
	return
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := r.ParseForm(); err != nil {
		_, _ = fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}
	email := r.FormValue(APIParameterKeyEmail)
	password := r.FormValue(APIParameterKeyPassword)

	success, data, authErr := authenticateUser(email, password)
	response := Response{
		Type:    APITypeLogin,
		Success: success,
		Data:    data,
		Error:   authErr,
	}
	writeResponse(w, response)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	response := Response{
		Type:    APITypeLogout,
		Success: true,
		Error: "",
	}
	writeResponse(w, response)
}

func randIn(low, hi int) int {
	return low + rand.Intn(hi-low)
}

func createRandomUsername(email string) string {
	parts := strings.Split(email, "@")
	return parts[0] + strconv.Itoa(randIn(1000, 9999))
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := r.ParseForm(); err != nil {
		_, _ = fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}
	email := r.FormValue(APIParameterKeyEmail)
	//check valid email or not
	//check email existed or not
	if _, ok := userAccounts[email]; ok {
		response := Response{
			Type:    APITypeRegister,
			Success: false,
			Error:   APIErrorUserExisted,
		}
		writeResponse(w, response)
		return
	}

	password := r.FormValue(APIParameterKeyPassword)
	// Set user account with password
	userAccounts[email] = password
	// Create random username and set to username list
	usernames[email] = createRandomUsername(email)

	// Create jwt
	_, data, _ := authenticateUser(email, password)
	response := Response{
		Type:    APITypeRegister,
		Success: true,
		Data:    data,
	}
	writeResponse(w, response)
}

func decodeJWT(jwt string) (JWT, error) {
	data, _ := b64.StdEncoding.DecodeString(jwt)
	jwtObject := &JWT{}
	err := json.Unmarshal(data, jwtObject)
	return *(jwtObject), err
}
func handleWSConnections(w http.ResponseWriter, r *http.Request) {
	jwt := r.URL.Query().Get("token")
	if jwt == "" {
		_, _ = w.Write([]byte("Wrong authentication"))
		return
	}
	jwtObject, err := decodeJWT(jwt)
	if err != nil {
		_, _ = w.Write([]byte("Wrong authentication"))
		return
	}
	if jwtObject.Exp <= time.Now().Unix() {
		_, _ = w.Write([]byte("Token expired"))
		return
	}

	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("error")
	}
	// Add to chain
	// Add conn to user conns
	username := jwtObject.Username
	userConns[username] = ws
	userOnlines <- username

	log.Printf("connected from: %s", ws.RemoteAddr().String())
	// Make sure we close the connection when the function returns
	defer ws.Close()

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Println("Read error:", err)
			delete(userConns, username)
			userProfiles[username].OnlineState = UserOnlineStateOffline
			// notify friends that this user going to offline
			msg := Message{
				Type: APITypeOnlineStateChange,
				Data: Data{
					Username:    username,
					OnlineState: UserOnlineStateOffline,
				},
			}
			//broadcast this message
			broadcast <- msg

			break
		}
		switch msg.Type {
		case APITypeOffer:
			conn := userConns[msg.Data.ToID]
			if conn != nil {
				log.Println("Sending offer to:", msg.Data.ToID)
				err := conn.WriteJSON(msg)
				if err != nil {
					log.Println("Write error:", err)
				}
			} else {
				log.Println("User", msg.Data.ToID, "not online")
				err := ws.WriteJSON(Message{
					Type: APITypeOfferResponse,
					Data: Data{
						FromID:  msg.Data.ToID,
						Success: false,
					},
				})
				if err != nil {
					log.Println("Write error:", err)
				}
			}
		case APITypeAnswer:
			conn := userConns[msg.Data.ToID]
			if conn != nil {
				log.Println("Sending answer to:", msg.Data.ToID)
				err := conn.WriteJSON(msg)
				if err != nil {
					log.Println("Write error:", err)
				}
			}
		case APITypeCandidate:
			//handle send candidate to user
			conn := userConns[msg.Data.ToID]
			if conn != nil {
				log.Println("Sending candidate to:", msg.Data.ToID)
				err := conn.WriteJSON(msg)
				if err != nil {
					log.Println("Write error:", err)
				}
			}
		case APITypeOnlineStateChange:
			broadcast <- msg
		default:
			log.Println("Error: Unexpected type: ", msg.Type)
		}
	}
}

func broadcastMessage() {
	for {
		msg := <-broadcast
		username := msg.Data.Username

		if friends, ok := userFriends[username]; ok {
			for _, friend := range friends {
				if friend.OnlineState != UserOnlineStateOffline {
					conn := userConns[friend.Username]
					err := conn.WriteJSON(msg)
					if err != nil {
						log.Printf("error: %v", err)
					}
				}
			}
		}
	}
}

func setOnlineUser() {
	for {
		username := <-userOnlines
		userProfiles[username].OnlineState = UserOnlineStateOnline
	}
}
