package routes

import (
	"database/sql"
	"net/http"

	"github.com/code-golf/code-golf/session"
	"github.com/code-golf/code-golf/trophy"
)

// GolferTrophies serves GET /golfers/{golfer}
func GolferTrophies(w http.ResponseWriter, r *http.Request) {
	golfer := session.GolferInfo(r).Golfer

	type EarnedTrophy struct {
		Count, Percent int
		Earned         sql.NullTime
		Trophy         *trophy.Trophy
	}

	data := struct {
		Max      int
		Trophies []EarnedTrophy
	}{
		Trophies: make([]EarnedTrophy, 0, len(trophy.List)),
	}

	tx, err := session.Database(r).BeginTx(
		r.Context(),
		&sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true},
	)
	if err != nil {
		panic(err)
	}

	defer tx.Rollback()

	if err := tx.QueryRow(
		"SELECT COUNT(DISTINCT user_id) FROM trophies",
	).Scan(&data.Max); err != nil {
		panic(err)
	}

	if data.Max == 0 {
		data.Max = 1
	}

	rows, err := tx.Query(
		`WITH count AS (
		    SELECT trophy, COUNT(user_id)
		      FROM (SELECT UNNEST(ENUM_RANGE(NULL::trophy)) trophy) x
		 LEFT JOIN trophies USING(trophy)
		  GROUP BY trophy
		), earned AS (
		    SELECT trophy, earned FROM trophies WHERE user_id = $1
		)  SELECT *
		     FROM count
		LEFT JOIN earned USING(trophy)
		 ORDER BY count DESC, trophy`,
		golfer.ID,
	)
	if err != nil {
		panic(err)
	}

	for rows.Next() {
		var earned EarnedTrophy
		var trophyID string

		if err := rows.Scan(
			&trophyID,
			&earned.Count,
			&earned.Earned,
		); err != nil {
			panic(err)
		}

		earned.Percent = earned.Count * 100 / data.Max
		earned.Trophy = trophy.ByID[trophyID]

		data.Trophies = append(data.Trophies, earned)
	}

	if err := rows.Err(); err != nil {
		panic(err)
	}

	if err := tx.Commit(); err != nil {
		panic(err)
	}

	render(w, r, "golfer/trophies", golfer.Name, data)
}
