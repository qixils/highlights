package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var categoryWeblink, _ = regexp.Compile("\\?h=([a-zA-Z0-9_]{1,100})")
var db *sql.DB

// TODO: context.WithCancel
var ctx context.Context = context.TODO()

type SrcTimes struct {
	Primary float64 `json:"primary_t"`
}

type SrcPlayerInfo struct {
	// "guest" or "user"
	Rel string `json:"rel"`
	// when rel is "user"
	Id *string `json:"id"`
	// when rel is "guest"
	Name *string `json:"name"`
}

type SrcRun struct {
	Id      string          `json:"id"`
	Times   SrcTimes        `json:"times"`
	Players []SrcPlayerInfo `json:"players"`
	Date    string          `json:"date"`
	Weblink string          `json:"weblink"`
}

type SrcLeaderboardRun struct {
	Place uint   `json:"place"`
	Run   SrcRun `json:"run"`
}

type SrcNames struct {
	International string  `json:"international"`
	Japanese      *string `json:"japanese"`
}

type SrcUri struct {
	Uri string `json:"uri"`
}

type SrcAssets struct {
	Icon          *SrcUri `json:"icon"`
	SupporterIcon *SrcUri `json:"supporterIcon"`
	Image         *SrcUri `json:"image"`
}

type SrcNameColor struct {
	// hex color codes
	Light string `json:"light"`
	Dark  string `json:"dark"`
}

type SrcNameStyle struct {
	// I believe this is always 'gradient'
	Style     string       `json:"style"`
	ColorFrom SrcNameColor `json:"color-from"`
	ColorTo   SrcNameColor `json:"color-to"`
}

type SrcPlayer struct {
	// "guest" or "user"
	Rel string `json:"rel"`
	// when rel is "guest"
	Name *string `json:"name"`
	// remainder are when rel is "user"
	Id     *string    `json:"id"`
	Names  *SrcNames  `json:"names"`
	Assets *SrcAssets `json:"assets"`
}

type SrcPlayerData struct {
	Data []SrcPlayer `json:"data"`
}

type SrcLeaderboardData struct {
	Timing  string              `json:"timing"`
	Runs    []SrcLeaderboardRun `json:"runs"`
	Players SrcPlayerData       `json:"players"`
}

type SrcLeaderboard struct {
	Data *SrcLeaderboardData `json:"data"`
}

type SrcGameData struct {
	Id           string `json:"id"`
	Abbreviation string `json:"abbreviation"`
}

type SrcGame struct {
	Data *SrcGameData `json:"data"`
}

type SrcCategory struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Weblink string `json:"weblink"`
}

type SrcCategories struct {
	Data []SrcCategory `json:"data"`
}

type UserStat struct {
	Name      string
	Picture   string
	Url       string
	Statistic int
}

type MonthlyStats struct {
	// human name of the game; for creating URLs
	Game     string
	Improved []UserStat
	Rising   []UserStat
}

type SearchResultCache interface {
	GetCachedAt() time.Time
}

type GameSearchResultCache struct {
	Game     *SrcGame
	CachedAt time.Time
}

type CategorySearchResultCache struct {
	Category *SrcCategory
	CachedAt time.Time
}

func (c GameSearchResultCache) GetCachedAt() time.Time {
	return c.CachedAt
}

func (c CategorySearchResultCache) GetCachedAt() time.Time {
	return c.CachedAt
}

func updateCache[T SearchResultCache](m map[string]T) {
	toDelete := make([]string, 0)
	for k, v := range m {
		if v.GetCachedAt().Before(time.Now().Add(time.Minute * -60)) {
			toDelete = append(toDelete, k)
		}
	}
	for _, k := range toDelete {
		delete(m, k)
	}
}

type GameCacheRequestItem struct {
	Key     string
	Value   *GameSearchResultCache
	Respond *chan *GameSearchResultCache
}

var gameCacheQueue chan GameCacheRequestItem = make(chan GameCacheRequestItem, 50)

func gameCacheRoutine() {
	gameCache := make(map[string]GameSearchResultCache)

	for {
		message := <-gameCacheQueue
		updateCache(gameCache)
		var result GameSearchResultCache
		var prs bool

		if message.Value != nil {
			gameCache[message.Key] = *message.Value
			result = *message.Value
			prs = true
		} else {
			result, prs = gameCache[message.Key]
		}

		if message.Respond != nil {
			if prs {
				*message.Respond <- &result
			} else {
				*message.Respond <- nil
			}
		}
	}
}

type CategoryCacheRequestItem struct {
	Key     string
	Value   *CategorySearchResultCache
	Respond *chan *CategorySearchResultCache
}

var categoryCacheQueue chan CategoryCacheRequestItem = make(chan CategoryCacheRequestItem, 50)

func categoryCacheRoutine() {
	categoryCache := make(map[string]CategorySearchResultCache)

	for {
		message := <-categoryCacheQueue
		updateCache(categoryCache)
		var result CategorySearchResultCache
		var prs bool

		if message.Value != nil {
			categoryCache[message.Key] = *message.Value
			result = *message.Value
			prs = true
		} else {
			result, prs = categoryCache[message.Key]
		}

		if message.Respond != nil {
			if prs {
				*message.Respond <- &result
			} else {
				*message.Respond <- nil
			}
		}
	}
}

/* https://go.dev/wiki/SQLInterface */
func getTable[T any](rows *sql.Rows) (out []T) {
	var table []T = make([]T, 0)
	for rows.Next() {
		var data T
		s := reflect.ValueOf(&data).Elem()
		numCols := s.NumField()
		columns := make([]any, numCols)

		for i := range numCols {
			field := s.Field(i)
			columns[i] = field.Addr().Interface()
		}

		if err := rows.Scan(columns...); err != nil {
			fmt.Println("Case Read Error ", err)
		}

		table = append(table, data)
	}
	return table
}

func readJson[T any](resp *http.Response, out *T) {
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	dec.Decode(out)
}

func highlights(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	ping := db.PingContext(ctx)
	if ping != nil {
		w.WriteHeader(http.StatusInternalServerError)
		// restart the process
		os.Exit(1)
	}

	inputGameId := req.URL.Query().Get("game")
	if len(inputGameId) == 0 {
		return
	}
	if len(inputGameId) > 100 {
		inputGameId = inputGameId[:100]
	}

	inputLeaderboardId := req.URL.Query().Get("leaderboard")
	if len(inputLeaderboardId) == 0 {
		return
	}
	if len(inputLeaderboardId) > 100 {
		inputLeaderboardId = inputLeaderboardId[:100]
	}

	// TODO: channel (avoid asybc map writing)
	gameBody := SrcGame{}
	categoryBody := SrcCategory{}

	gameReceiver := make(chan *GameSearchResultCache, 1)
	gameCacheQueue <- GameCacheRequestItem{Key: inputGameId, Respond: &gameReceiver}
	var cachedGame *GameSearchResultCache
	select {
	case res := <-gameReceiver:
		cachedGame = res
	case <-time.After(3 * time.Second):
		cachedGame = nil
	}

	if cachedGame != nil {
		if cachedGame.Game == nil {
			return
		}
		gameBody = *cachedGame.Game
	} else {
		// get game data
		gameResp, gameErr := http.Get("https://www.speedrun.com/api/v1/games/" + inputGameId)
		if gameErr != nil {
			gameCacheQueue <- GameCacheRequestItem{Key: inputGameId, Value: &GameSearchResultCache{CachedAt: time.Now()}}
			return
		}
		readJson(gameResp, &gameBody)
		gameCacheQueue <- GameCacheRequestItem{Key: inputGameId, Value: &GameSearchResultCache{CachedAt: time.Now(), Game: &gameBody}}
	}

	if gameBody.Data == nil {
		return
	}
	gameId := gameBody.Data.Id

	leaderboardCacheKey := gameId + ":" + inputLeaderboardId
	categoryReceiver := make(chan *CategorySearchResultCache, 1)
	categoryCacheQueue <- CategoryCacheRequestItem{Key: leaderboardCacheKey, Respond: &categoryReceiver}
	var cachedCategory *CategorySearchResultCache
	select {
	case res := <-categoryReceiver:
		cachedCategory = res
	case <-time.After(3 * time.Second):
		cachedCategory = nil
	}
	if cachedCategory != nil {
		if cachedCategory.Category == nil {
			return
		}
		categoryBody = *cachedCategory.Category
	} else {
		// get category data
		categoriesResp, categoriesErr := http.Get("https://www.speedrun.com/api/v1/games/" + inputGameId + "/categories")
		if categoriesErr != nil {
			// TODO wrong keys on all of these
			categoryCacheQueue <- CategoryCacheRequestItem{Key: leaderboardCacheKey, Value: &CategorySearchResultCache{CachedAt: time.Now()}}
			return
		}
		categories := SrcCategories{}
		readJson(categoriesResp, &categories)
		if categories.Data == nil {
			categoryCacheQueue <- CategoryCacheRequestItem{Key: leaderboardCacheKey, Value: &CategorySearchResultCache{CachedAt: time.Now()}}
			return
		}

		matched := false
		for _, category := range categories.Data {
			matches := categoryWeblink.FindStringSubmatch(category.Weblink)
			if len(matches) < 2 {
				continue
			}
			if inputLeaderboardId == matches[1] {
				matched = true
				categoryBody = category
				categoryCacheQueue <- CategoryCacheRequestItem{Key: leaderboardCacheKey, Value: &CategorySearchResultCache{CachedAt: time.Now(), Category: &category}}
				break
			}
		}

		if !matched {
			categoryCacheQueue <- CategoryCacheRequestItem{Key: leaderboardCacheKey, Value: &CategorySearchResultCache{CachedAt: time.Now()}}
			return
		}
	}
	if categoryBody.Id == "" {
		return
	}
	leaderboardId := categoryBody.Id

	// compute month to calculate data for
	now := time.Now().UTC()
	targetYear := now.Year()
	targetMonth := now.Month() - 1
	if targetMonth < time.January {
		targetYear -= 1
		targetMonth = time.December
	}

	targetMonthStart := time.Date(targetYear, targetMonth, 1, 0, 0, 0, 0, time.UTC)
	targetMonthEnd := targetMonthStart.AddDate(0, 1, 0).Add(-24 * time.Hour)

	prevYear := targetYear
	prevMonth := targetMonth - 1
	if prevMonth < time.January {
		prevYear -= 1
		prevMonth = time.December
	}
	prevMonthStart := time.Date(prevYear, prevMonth, 1, 0, 0, 0, 0, time.UTC)
	prevMonthEnd := prevMonthStart.AddDate(0, 1, 0).Add(-24 * time.Hour)

	var savedYear = 0
	var savedMonth time.Month = 0
	err := db.QueryRowContext(ctx, "SELECT Year, Month FROM Leaderboards WHERE Game = ? AND Category = ?", gameId, leaderboardId).Scan(&savedYear, &savedMonth)

	if savedYear < targetYear || savedMonth < targetMonth {
		startYear := targetYear
		startMonth := targetMonth - 1
		if startMonth < time.January {
			startYear -= 1
			startMonth = time.December
		}

		for {
			if savedYear < startYear || savedMonth < startMonth {
				nextYear := startYear
				nextMonth := startMonth + 1
				if nextMonth > time.December {
					nextYear += 1
					nextMonth = time.January
				}
				endDate := time.Date(nextYear, nextMonth, 1, 12, 0, 0, 0, time.UTC).Add(-24 * time.Hour)
				endDateISO := fmt.Sprintf("%d-%02d-%02d", endDate.Year(), endDate.Month(), endDate.Day())

				resp, err := http.Get("https://www.speedrun.com/api/v1/leaderboards/" + gameId + "/category/" + leaderboardId + "?embed=players&date=" + endDateISO)
				if err != nil {
					return
				}
				body := SrcLeaderboard{}
				readJson(resp, &body)

				txn, err := db.BeginTx(context.TODO(), nil)
				if err != nil {
					return
				}

				playersData := map[string]SrcPlayer{}
				for _, player := range body.Data.Players.Data {
					if player.Id == nil {
						continue
					}
					playersData[*player.Id] = player
				}

				for _, run := range body.Data.Runs {
					runDate, err := time.Parse("2006-01-02", run.Run.Date)
					if err != nil {
						continue
					}
					txn.Exec("INSERT INTO Runs (Run, Time, Date, Game, Category) VALUES (?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE Time = VALUES(Time), Date = VALUES(Date), Game = VALUES(Game), Category = VALUES(Category)", run.Run.Id, uint(math.Ceil(run.Run.Times.Primary)), runDate, gameId, leaderboardId)
					for _, player := range run.Run.Players {
						var playerId string
						if player.Id != nil {
							playerId = *player.Id
						} else if player.Name != nil {
							playerId = *player.Name
						} else {
							continue
						}
						txn.Exec("INSERT INTO Runners (Run, Player) VALUES (?, ?)", run.Run.Id, playerId)
						playerData, prs := playersData[playerId]
						if prs {
							icon := ""
							if playerData.Assets != nil && playerData.Assets.Image != nil {
								icon = playerData.Assets.Image.Uri
							}
							txn.Exec("INSERT INTO Players (Player, Name, Icon) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE Name = VALUES(Name), Icon = VALUES(Icon)", playerId, playerData.Names.International, icon)
						}
					}
				}

				txn.Exec("INSERT INTO Leaderboards (Game, Category, Year, Month) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE Year = VALUES(Year), Month = VALUES(Month)", gameId, leaderboardId, startYear, startMonth)
				err = txn.Commit()
				if err != nil {
					return
				}
			}

			if startMonth == targetMonth {
				break
			} else {
				startYear = targetYear
				startMonth = targetMonth
			}
		}
	}

	// todo: variables

	targetMonthStartFmt := targetMonthStart.Format("2006-01-02")
	targetMonthEndFmt := targetMonthEnd.Format("2006-01-02")
	prevMonthStartFmt := prevMonthStart.Format("2006-01-02")
	prevMonthEndFmt := prevMonthEnd.Format("2006-01-02")

	improved_rows, err := db.QueryContext(ctx, `
		WITH curr AS (
			SELECT r.Player, ru.Run, ru.Time AS curr_pb,
			       ROW_NUMBER() OVER (PARTITION BY r.Player ORDER BY ru.Time ASC, ru.Run ASC) AS rn
			FROM Runners r
			JOIN Runs ru ON r.Run = ru.Run
			WHERE ru.Game = ? AND ru.Category = ?
			  AND ru.Date BETWEEN ? AND ?
		), best_curr AS (
			SELECT Player, Run, curr_pb
			FROM curr
			WHERE rn = 1
		), prev AS (
			SELECT r.Player, MIN(ru.Time) AS prev_pb
			FROM Runners r
			JOIN Runs ru ON r.Run = ru.Run
			WHERE ru.Game = ? AND ru.Category = ?
			  AND ru.Date BETWEEN ? AND ?
			GROUP BY r.Player
		)
		SELECT p.Name, p.Icon, best_curr.Run, (prev.prev_pb - best_curr.curr_pb) AS Statistic
		FROM Players p
		JOIN best_curr ON p.Player = best_curr.Player
		JOIN prev ON prev.Player = best_curr.Player
		ORDER BY Statistic DESC
		LIMIT 5`, gameId, leaderboardId, targetMonthStartFmt, targetMonthEndFmt, gameId, leaderboardId, prevMonthStartFmt, prevMonthEndFmt)
	if err != nil {
		return
	}
	improved_table := getTable[UserStat](improved_rows)

	rising_rows, err := db.QueryContext(ctx, `
		WITH curr AS (
			SELECT r.Player, ru.Run, ru.Time AS curr_pb,
			       ROW_NUMBER() OVER (PARTITION BY r.Player ORDER BY ru.Time ASC, ru.Run ASC) AS rn
			FROM Runners r
			JOIN Runs ru ON r.Run = ru.Run
			WHERE ru.Game = ? AND ru.Category = ?
			  AND ru.Date BETWEEN ? AND ?
		), best_curr AS (
			SELECT Player, Run, curr_pb
			FROM curr
			WHERE rn = 1
		), prior AS (
			SELECT DISTINCT r.Player
			FROM Runners r
			JOIN Runs ru ON r.Run = ru.Run
			WHERE ru.Game = ? AND ru.Category = ?
			  AND ru.Date < ?
		)
		SELECT p.Name, p.Icon, best_curr.Run, best_curr.curr_pb AS Statistic
		FROM Players p
		JOIN best_curr ON p.Player = best_curr.Player
		LEFT JOIN prior ON prior.Player = best_curr.Player
		WHERE prior.Player IS NULL
		ORDER BY Statistic ASC
		LIMIT 5`, gameId, leaderboardId, targetMonthStartFmt, targetMonthEndFmt, gameId, leaderboardId, targetMonthStartFmt)
	if err != nil {
		return
	}
	rising_table := getTable[UserStat](rising_rows)

	table := MonthlyStats{Improved: improved_table, Rising: rising_table, Game: gameBody.Data.Abbreviation}

	enc := json.NewEncoder(w)
	enc.Encode(table)
}

func main() {

	var err error
	db, err = sql.Open("mysql", os.Getenv("MYSQL_DSN")) // "root:password@/speedclub_highlights"
	if err != nil {
		log.Fatal(err)
		return
	}
	if err := db.PingContext(ctx); err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	go gameCacheRoutine()
	go categoryCacheRoutine()

	db.Exec("CREATE TABLE IF NOT EXISTS Players ( Player CHAR(8) PRIMARY KEY, Name TEXT NOT NULL, Icon TEXT )")
	db.Exec("CREATE TABLE IF NOT EXISTS Runs ( Run CHAR(8) PRIMARY KEY, Time INT NOT NULL, Date DATE NOT NULL, Game CHAR(8) NOT NULL, Category CHAR(8) NOT NULL )")
	db.Exec("CREATE TABLE IF NOT EXISTS Runners ( Run CHAR(8) NOT NULL, Player TEXT NOT NULL, FOREIGN KEY (Run) REFERENCES Runs(Run), UNIQUE KEY RunPlayerID (Run,Player) )")
	db.Exec("CREATE TABLE IF NOT EXISTS Leaderboards ( Game CHAR(8) NOT NULL, Category CHAR(8) NOT NULL, Year SMALLINT NOT NULL, Month TINYINT NOT NULL, UNIQUE KEY GameCategoryID (Game,Category) )")
	// todo: indexes?

	http.HandleFunc("GET /api/v1/highlights", highlights)

	http.ListenAndServe(":30100", nil)
}
