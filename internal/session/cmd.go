package session

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"SetSecrets/internal/enroll"
	"SetSecrets/internal/prompt"
	gamesession "SetSecrets/internal/proto/gamesession"
	"SetSecrets/internal/target"
	"SetSecrets/internal/tpm"

	"github.com/spf13/cobra"
)

// SessionFile represents the saved session state.
type SessionFile struct {
	SessionID string `json:"sessionId"`
}

// resolveConfig resolves API URL, CA file, and device ID based on environment and flags.
func resolveConfig(envName, customUrl, customCa, customDevice string) (string, string, string) {
	backend := target.Current()
	apiUrl := backend.ApiUrl
	caFile := ""

	switch strings.ToLower(envName) {
	case "staging":
		apiUrl = backend.ApiUrl
		if backend.UseTls {
			caFile = "root_ca.crt"
		}
	case "prod", "production":
		apiUrl = "https://api.worldpinball.com"
		caFile = ""
	default: // dev
		apiUrl = backend.ApiUrl
		caFile = ""
	}

	if customUrl != "" {
		apiUrl = customUrl
	}
	if customCa != "" {
		caFile = customCa
	}

	deviceId := customDevice
	if deviceId == "" {
		if cfg, err := enroll.LoadConfigFile("enroll.json"); err == nil && cfg.DeviceId != "" {
			deviceId = cfg.DeviceId
			fmt.Printf("ℹ Auto-detected Device ID from enroll.json: %s\n", deviceId)
		}
	}

	return apiUrl, caFile, deviceId
}

// resolveGrpcTarget resolves gRPC target URL and TLS flag based on target backend.
func resolveGrpcTarget(customUrl string) (string, bool) {
	backend := target.Current()
	if customUrl != "" {
		useTls := strings.HasPrefix(strings.ToLower(customUrl), "https://")
		return customUrl, useTls
	}
	return backend.GrpcUrl, backend.UseTls
}

// resolveSessionID returns the explicit sessionID if provided, otherwise reads it from session.json.
func resolveSessionID(customSessionID string) string {
	if customSessionID != "" {
		return customSessionID
	}
	data, err := os.ReadFile("session.json")
	if err == nil {
		var s SessionFile
		if err := json.Unmarshal(data, &s); err == nil && s.SessionID != "" {
			fmt.Printf("ℹ Auto-detected Session ID from session.json: %s\n", s.SessionID)
			return s.SessionID
		}
	}
	return ""
}

// saveSessionID writes the active session ID to session.json.
func saveSessionID(sessionID string) error {
	data, err := json.MarshalIndent(SessionFile{SessionID: sessionID}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("session.json", data, 0644)
}

// parseScores parses score strings like '1:2500000,2:1800000' or single score/slot flags.
func parseScores(scoresStr string, singleScore int64, singleSlot int) ([]SlotScoreRequest, error) {
	if strings.TrimSpace(scoresStr) == "" {
		return []SlotScoreRequest{
			{
				SlotNumber: singleSlot,
				Score:      singleScore,
			},
		}, nil
	}

	parts := strings.Split(scoresStr, ",")
	var result []SlotScoreRequest
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, ":") {
			kv := strings.SplitN(part, ":", 2)
			slot, err := strconv.Atoi(strings.TrimSpace(kv[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid slot in '%s': %w", part, err)
			}
			score, err := strconv.ParseInt(strings.TrimSpace(kv[1]), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid score in '%s': %w", part, err)
			}
			result = append(result, SlotScoreRequest{SlotNumber: slot, Score: score})
		} else {
			score, err := strconv.ParseInt(part, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid score '%s': %w", part, err)
			}
			result = append(result, SlotScoreRequest{SlotNumber: i + 1, Score: score})
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no valid scores found in --scores '%s'", scoresStr)
	}
	return result, nil
}

// NewStartCmd creates the 'start' command to initiate an authenticated game session.
func NewStartCmd() *cobra.Command {
	var (
		startDeviceId string
		startMode     string
		startApiUrl   string
		startEnv      string
		startKeyFile  string
		startCertFile string
		startCaFile   string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"create-game", "start"},
		Short:   "Create a new game session lobby",
		Run: func(cmd *cobra.Command, args []string) {
			apiUrl, caFile, deviceId := resolveConfig(startEnv, startApiUrl, startCaFile, startDeviceId)
			if deviceId == "" {
				fmt.Println("Error: --device-id is required (or provide enroll.json)")
				os.Exit(1)
			}

			machine := tpm.NewSimulatedMachine()
			defer func() {
				_ = machine.Close()
			}()
			if err := machine.LoadKeyFromFile(startKeyFile); err != nil {
				fmt.Printf("Error loading private key from %s: %v\n", startKeyFile, err)
				os.Exit(1)
			}
			fmt.Printf("[%s] Signing canonical request and contacting server (%s)...\n", strings.ToUpper(startEnv), apiUrl)
			resp, err := StartSession(apiUrl, deviceId, machine, startMode, startCertFile, startKeyFile, caFile)
			if err != nil {
				fmt.Printf("Failed to create session: %v\n", err)
				os.Exit(1)
			}
			_ = saveSessionID(resp.SessionID)
			fmt.Printf("✔ Game session created successfully! Lobby is open.\n")
			fmt.Printf("  Session ID  : %s\n", resp.SessionID)
			fmt.Printf("  Server Time : %d\n", resp.ServerTime)
		},
	}

	cmd.Flags().StringVar(&startEnv, "env", "dev", "Target environment profile: dev, staging, prod")
	cmd.Flags().StringVar(&startApiUrl, "url", "", "Override API base URL")
	cmd.Flags().StringVar(&startDeviceId, "device-id", "", "Device GUID registered on the server (falls back to enroll.json)")
	cmd.Flags().StringVar(&startMode, "mode", "mode-1", "Game mode code")
	cmd.Flags().StringVar(&startKeyFile, "key", "device.key", "Path to the private key PEM file")
	cmd.Flags().StringVar(&startCertFile, "cert", "device.crt", "Path to the X.509 certificate file")
	cmd.Flags().StringVar(&startCaFile, "ca", "", "Path to the root CA certificate file (override)")

	return cmd
}

// NewSubmitCmd creates the 'submit' command to submit scores and close a game session.
func NewSubmitCmd() *cobra.Command {
	var (
		submitDeviceId  string
		submitSessionId string
		submitScore     int64
		submitSlot      int
		submitScores    string
		submitApiUrl    string
		submitEnv       string
		submitKeyFile   string
		submitCertFile  string
		submitCaFile    string
	)

	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Submit player score and end the game session",
		Run: func(cmd *cobra.Command, args []string) {
			apiUrl, caFile, deviceId := resolveConfig(submitEnv, submitApiUrl, submitCaFile, submitDeviceId)
			submitSessionId = resolveSessionID(submitSessionId)
			if deviceId == "" || submitSessionId == "" {
				fmt.Println("Error: --session-id is required (and --device-id or enroll.json)")
				os.Exit(1)
			}

			var scores []SlotScoreRequest

			// 1. User provided explicit --scores flag
			if submitScores != "" {
				s, err := parseScores(submitScores, submitScore, submitSlot)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
				scores = s
			} else {
				// 2. Smart auto-detection: query live lobby to find active players
				lobbyResp, err := GetLobby(cmd.Context(), apiUrl, true, submitSessionId)
				if err == nil && lobbyResp != nil {
					// Read candidate scores from players.json if available
					playerScores := make(map[int32]int64)
					if data, err := os.ReadFile("players.json"); err == nil {
						var candidates []prompt.PlayerCandidate
						if err := json.Unmarshal(data, &candidates); err == nil {
							for _, c := range candidates {
								if c.Score > 0 {
									playerScores[c.Slot] = c.Score
								}
							}
						}
					}

					for _, slot := range lobbyResp.Slots {
						if slot.IsOccupied {
							scoreVal := int64(1500000 + int64(slot.SlotNumber)*350000)
							if val, ok := playerScores[slot.SlotNumber]; ok {
								scoreVal = val
							}
							scores = append(scores, SlotScoreRequest{
								SlotNumber: int(slot.SlotNumber),
								Score:      scoreVal,
							})
						}
					}

					if len(scores) > 0 {
						fmt.Printf("ℹ Auto-detected %d active player(s) in lobby for score submission.\n", len(scores))
					}
				}

				// 3. Fallback to default single player if no lobby detected
				if len(scores) == 0 {
					s, err := parseScores(submitScores, submitScore, submitSlot)
					if err != nil {
						fmt.Printf("Error: %v\n", err)
						os.Exit(1)
					}
					scores = s
				}
			}

			machine := tpm.NewSimulatedMachine()
			defer func() {
				_ = machine.Close()
			}()
			if err := machine.LoadKeyFromFile(submitKeyFile); err != nil {
				fmt.Printf("Error loading private key from %s: %v\n", submitKeyFile, err)
				os.Exit(1)
			}
			fmt.Printf("[%s] Signing score submission and contacting server (%s)...\n", strings.ToUpper(submitEnv), apiUrl)
			resp, err := SubmitScore(apiUrl, deviceId, machine, submitSessionId, scores, submitCertFile, submitKeyFile, caFile)
			if err != nil {
				fmt.Printf("Failed to submit score: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✔ Scores submitted successfully (%d player(s))!\n", len(resp.ScoreEntryIDs))
			for i, id := range resp.ScoreEntryIDs {
				if i < len(scores) {
					fmt.Printf("  • Slot %d | Score: %d -> Entry ID: %s\n", scores[i].SlotNumber, scores[i].Score, id)
				} else {
					fmt.Printf("  • Entry ID: %s\n", id)
				}
			}
		},
	}

	cmd.Flags().StringVar(&submitEnv, "env", "dev", "Target environment profile: dev, staging, prod")
	cmd.Flags().StringVar(&submitApiUrl, "url", "", "Override API base URL")
	cmd.Flags().StringVar(&submitDeviceId, "device-id", "", "Device GUID registered on the server (falls back to enroll.json)")
	cmd.Flags().StringVar(&submitSessionId, "session-id", "", "Game Session GUID")
	cmd.Flags().Int64Var(&submitScore, "score", 1250000, "Single player final score to submit (default when --scores is not set)")
	cmd.Flags().IntVar(&submitSlot, "slot", 1, "Single player slot number 1-4 (default when --scores is not set)")
	cmd.Flags().StringVar(&submitScores, "scores", "", "Comma-separated list of scores or slot:score (e.g. '1:2500000,2:1800000' or '2500000,1800000')")
	cmd.Flags().StringVar(&submitKeyFile, "key", "device.key", "Path to the private key PEM file")
	cmd.Flags().StringVar(&submitCertFile, "cert", "device.crt", "Path to the X.509 certificate file")
	cmd.Flags().StringVar(&submitCaFile, "ca", "", "Path to the root CA certificate file (override)")

	return cmd
}

// PrintLobby renders the 4 slots and state in a readable terminal format.
func PrintLobby(resp *gamesession.SessionLobbyResponse) {
	fmt.Printf("\n Session : %s | State: %s\n", resp.SessionId, resp.SessionState)
	fmt.Println("───────────────────────────────────────────────────")
	for _, slot := range resp.Slots {
		status := "🟢 Occupied"
		details := ""
		if !slot.IsOccupied {
			status = "⚪ Empty"
		} else if slot.UserId != nil {
			details = fmt.Sprintf("User: %s", *slot.UserId)
		} else if slot.GuestInitials != nil {
			details = fmt.Sprintf("Guest: %s", *slot.GuestInitials)
		}
		fmt.Printf("  Slot %d : %-12s %s\n", slot.SlotNumber, status, details)
	}
	fmt.Println("───────────────────────────────────────────────────")
}

// NewLobbyCmd creates the 'lobby' Cobra command and its subcommands (get, kick, watch).
func NewLobbyCmd() *cobra.Command {
	var (
		sessionID string
		slot      int32
		reason    string
		url       string
	)

	cmd := &cobra.Command{
		Use:   "lobby",
		Short: "Inspect and manage game session lobby slots (get, kick, watch)",
	}

	cmd.PersistentFlags().StringVar(&sessionID, "session-id", "", "Game Session GUID")
	cmd.PersistentFlags().StringVar(&url, "url", "", "gRPC Server URL (defaults to active backend target)")

	// Subcommand: get
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get current lobby status and slots",
		Run: func(c *cobra.Command, args []string) {
			sid := resolveSessionID(sessionID)
			if sid == "" {
				fmt.Println("Error: --session-id is required (or run 'wp-tools create' first)")
				os.Exit(1)
			}
			grpcTarget, useTls := resolveGrpcTarget(url)
			resp, err := GetLobby(c.Context(), grpcTarget, useTls, sid)
			if err != nil {
				fmt.Printf("Failed to get lobby: %v\n", err)
				os.Exit(1)
			}
			PrintLobby(resp)
		},
	}

	// Subcommand: kick
	kickCmd := &cobra.Command{
		Use:   "kick",
		Short: "Kick a player from a slot",
		Run: func(c *cobra.Command, args []string) {
			sid := resolveSessionID(sessionID)
			if sid == "" || slot < 1 || slot > 4 {
				fmt.Println("Error: --session-id and a valid --slot (1-4) are required")
				os.Exit(1)
			}
			grpcTarget, useTls := resolveGrpcTarget(url)
			resp, err := KickPlayer(c.Context(), grpcTarget, useTls, sid, slot, reason)
			if err != nil {
				fmt.Printf("Failed to kick player: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✔ Player in slot %d kicked successfully (%s)\n", resp.SlotNumber, resp.Message)
		},
	}
	kickCmd.Flags().Int32Var(&slot, "slot", 0, "Slot number to kick (1-4)")
	kickCmd.Flags().StringVar(&reason, "reason", "", "Optional kick reason")

	// Subcommand: watch
	watchCmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch lobby updates in real-time",
		Run: func(c *cobra.Command, args []string) {
			sid := resolveSessionID(sessionID)
			if sid == "" {
				fmt.Println("Error: --session-id is required (or run 'wp-tools create' first)")
				os.Exit(1)
			}
			grpcTarget, useTls := resolveGrpcTarget(url)
			fmt.Printf("[Watch] Watching lobby for session %s (Ctrl+C to quit)...\n", sid)
			err := WatchLobby(c.Context(), grpcTarget, useTls, sid, func(resp *gamesession.SessionLobbyResponse) {
				PrintLobby(resp)
			})
			if err != nil {
				fmt.Printf("Stream ended with error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("🏁 Lobby watch completed (session started or closed).")
		},
	}

	// 4. Subcommand: launch (or start)
	launchCmd := &cobra.Command{
		Use:     "launch",
		Aliases: []string{"start"},
		Short:   "Launch / start the game session, moving it from lobby to running",
		Run: func(c *cobra.Command, args []string) {
			sid := resolveSessionID(sessionID)
			if sid == "" {
				fmt.Println("Error: --session-id is required (or run 'wp-tools create' first)")
				os.Exit(1)
			}
			grpcTarget, useTls := resolveGrpcTarget(url)
			resp, err := LaunchSession(c.Context(), grpcTarget, useTls, sid)
			if err != nil {
				fmt.Printf("Failed to launch session: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("🚀 Session %s launched successfully! The game has started.\n", resp.SessionId)
		},
	}

	// Subcommand: add (flags or interactive TUI)
	var (
		addSlot     int32
		addInitials string
		addFile     string
	)

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add player(s) to the lobby (interactive TUI or flags)",
		Run: func(c *cobra.Command, args []string) {
			sid := resolveSessionID(sessionID)
			if sid == "" {
				fmt.Println("Error: --session-id is required (or run 'wp-tools create' first)")
				os.Exit(1)
			}
			grpcTarget, useTls := resolveGrpcTarget(url)

			// 1. Direct flags (--slot and --initials)
			if addSlot >= 1 && addSlot <= 4 && addInitials != "" {
				resp, err := AddPlayer(c.Context(), grpcTarget, useTls, sid, addSlot, addInitials)
				if err != nil {
					fmt.Printf("Failed to add player: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("✔ Player '%s' added to slot %d successfully (%s)\n", addInitials, resp.SlotNumber, resp.Message)
				return
			}

			// 2. Interactive TUI via players.json
			filePath := "players.json"
			if addFile != "" {
				filePath = addFile
			}

			var candidates []prompt.PlayerCandidate
			if data, err := os.ReadFile(filePath); err == nil {
				_ = json.Unmarshal(data, &candidates)
			}

			if len(candidates) > 0 {
				selected, err := prompt.SelectPlayersMulti(candidates)
				if err != nil {
					fmt.Printf("Prompt error: %v\n", err)
					os.Exit(1)
				}
				if len(selected) == 0 {
					fmt.Println("ℹ No players selected.")
					return
				}
				for _, p := range selected {
					resp, err := AddPlayer(c.Context(), grpcTarget, useTls, sid, p.Slot, p.Initials)
					if err != nil {
						fmt.Printf("❌ Failed to add slot %d (%s): %v\n", p.Slot, p.Initials, err)
					} else {
						fmt.Printf("✔ Player '%s' added to slot %d successfully!\n", p.Initials, resp.SlotNumber)
					}
				}
				return
			}

			fmt.Println("Please provide --slot and --initials (e.g. wp-tools lobby add --slot 2 --initials ALX) or create players.json")
		},
	}
	addCmd.Flags().Int32Var(&addSlot, "slot", 0, "Slot number (1-4)")
	addCmd.Flags().StringVar(&addInitials, "initials", "", "Guest initials (e.g. ALX)")
	addCmd.Flags().StringVar(&addFile, "file", "", "Path to players.json (optional)")

	cmd.AddCommand(addCmd)
	cmd.AddCommand(getCmd)
	cmd.AddCommand(kickCmd)
	cmd.AddCommand(watchCmd)
	cmd.AddCommand(launchCmd)

	return cmd
}
