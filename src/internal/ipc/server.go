//go:build windows

package ipc

import (
	"encoding/json"
	"log"
	"net"

	"veda-anchor-engine/src/api"

	"github.com/Microsoft/go-winio"
)

type Server struct {
	apiServer *api.Server
}

func NewServer(apiServer *api.Server) *Server {
	return &Server{apiServer: apiServer}
}

func (s *Server) Start() error {
	address := GetIPCAddress()

	config := &winio.PipeConfig{
		SecurityDescriptor: "D:(A;;GA;;;SY)(A;;GA;;;BA)(A;;GA;;;AU)",
	}
	listener, err := winio.ListenPipe(address, config)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("IPC Server listening on named pipe: %s", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var req Request
		if err := decoder.Decode(&req); err != nil {
			return
		}

		resp := s.dispatch(req)
		if err := encoder.Encode(resp); err != nil {
			log.Printf("Error encoding response: %v", err)
			return
		}
	}
}

func (s *Server) dispatch(req Request) Response {
	var result any
	var err error

	switch req.Method {
	case "Ping":
		result = true

	// --- Stats ---

	case "GetAppLeaderboard":
		var params struct {
			Since string `json:"since"`
			Until string `json:"until"`
		}
		json.Unmarshal(req.Params, &params)
		result, err = s.apiServer.GetAppLeaderboard(params.Since, params.Until)

	case "GetScreenTime":
		result, err = s.apiServer.GetScreenTime()

	case "GetTotalScreenTime":
		result, err = s.apiServer.GetTotalScreenTime()

	case "GetWebLeaderboard":
		var params struct {
			Since string `json:"since"`
			Until string `json:"until"`
		}
		json.Unmarshal(req.Params, &params)
		result, err = s.apiServer.GetWebLeaderboard(params.Since, params.Until)

	case "Search":
		var params struct {
			Query string `json:"query"`
			Since string `json:"since"`
			Until string `json:"until"`
		}
		json.Unmarshal(req.Params, &params)
		result, err = s.apiServer.Search(params.Query, params.Since, params.Until)

	case "GetWebLogs":
		var params struct {
			Query string `json:"query"`
			Since string `json:"since"`
			Until string `json:"until"`
		}
		json.Unmarshal(req.Params, &params)
		result, err = s.apiServer.GetWebLogs(params.Query, params.Since, params.Until)

	// --- App Blocklist ---

	case "GetAppBlocklist":
		result, err = s.apiServer.GetAppBlocklist()

	case "BlockApps":
		var names []string
		json.Unmarshal(req.Params, &names)
		err = s.apiServer.BlockApps(names)

	case "UnblockApps":
		var names []string
		json.Unmarshal(req.Params, &names)
		err = s.apiServer.UnblockApps(names)

	case "ClearAppBlocklist":
		err = s.apiServer.ClearAppBlocklist()

	case "SaveAppBlocklist":
		result, err = s.apiServer.SaveAppBlocklist()

	case "LoadAppBlocklist":
		var content []byte
		json.Unmarshal(req.Params, &content)
		err = s.apiServer.LoadAppBlocklist(content)

	// --- Web Blocklist ---

	case "GetWebBlocklist":
		result, err = s.apiServer.GetWebBlocklist()

	case "AddWebBlocklist":
		var domain string
		json.Unmarshal(req.Params, &domain)
		err = s.apiServer.AddWebBlocklist(domain)

	case "RemoveWebBlocklist":
		var domain string
		json.Unmarshal(req.Params, &domain)
		err = s.apiServer.RemoveWebBlocklist(domain)

	case "ClearWebBlocklist":
		err = s.apiServer.ClearWebBlocklist()

	case "SaveWebBlocklist":
		result, err = s.apiServer.SaveWebBlocklist()

	case "LoadWebBlocklist":
		var content []byte
		json.Unmarshal(req.Params, &content)
		err = s.apiServer.LoadWebBlocklist(content)

	// --- Auth ---

	case "GetIsAuthenticated":
		result = s.apiServer.GetIsAuthenticated()

	case "Logout":
		s.apiServer.Logout()
		result = true

	case "HasPassword":
		result, err = s.apiServer.HasPassword()

	case "Login":
		var params struct {
			Password string `json:"password"`
		}
		json.Unmarshal(req.Params, &params)
		result, err = s.apiServer.Login(params.Password)

	case "SetPassword":
		var params struct {
			Password string `json:"password"`
		}
		json.Unmarshal(req.Params, &params)
		err = s.apiServer.SetPassword(params.Password)

	// --- System ---

	case "Shutdown":
		s.apiServer.Shutdown()
		result = true

	case "Uninstall":
		var params struct {
			Password string `json:"password"`
		}
		json.Unmarshal(req.Params, &params)
		err = s.apiServer.Uninstall(params.Password)

	case "GetAutostartStatus":
		result, err = s.apiServer.GetAutostartStatus()

	case "EnableAutostart":
		err = s.apiServer.EnableAutostart()

	case "DisableAutostart":
		err = s.apiServer.DisableAutostart()

	case "ClearAppHistory":
		var params struct {
			Password string `json:"password"`
		}
		json.Unmarshal(req.Params, &params)
		err = s.apiServer.ClearAppHistory(params.Password)

	case "ClearWebHistory":
		var params struct {
			Password string `json:"password"`
		}
		json.Unmarshal(req.Params, &params)
		err = s.apiServer.ClearWebHistory(params.Password)

	// --- Agent Communication ---

	case "UpdateScreenTime":
		var params struct {
			PID     uint32 `json:"pid"`
			Seconds int64  `json:"seconds"`
		}
		json.Unmarshal(req.Params, &params)
		err = s.apiServer.UpdateScreenTime(params.PID, params.Seconds)

	case "ReportActiveApp":
		var params struct {
			PID     uint32 `json:"pid"`
			ExePath string `json:"exePath"`
		}
		json.Unmarshal(req.Params, &params)
		err = s.apiServer.ReportActiveApp(params.PID, params.ExePath)

	case "LogWebEvent":
		var params struct {
			URL       string `json:"url"`
			Title     string `json:"title"`
			VisitTime int64  `json:"visitTime"`
		}
		json.Unmarshal(req.Params, &params)
		err = s.apiServer.LogWebEvent(params.URL, params.Title, params.VisitTime)

	case "SaveWebMetadata":
		var params struct {
			Domain  string `json:"domain"`
			Title   string `json:"title"`
			IconURL string `json:"iconURL"`
		}
		json.Unmarshal(req.Params, &params)
		err = s.apiServer.SaveWebMetadata(params.Domain, params.Title, params.IconURL)

	default:
		return Response{ID: req.ID, Error: "Unknown method: " + req.Method}
	}

	if err != nil {
		return Response{ID: req.ID, Error: err.Error()}
	}

	return Response{ID: req.ID, Result: result}
}
