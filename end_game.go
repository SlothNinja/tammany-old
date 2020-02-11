package tammany

import (
	"encoding/gob"
	"fmt"
	"html/template"

	"bitbucket.org/SlothNinja/slothninja-games/sn/contest"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"bitbucket.org/SlothNinja/slothninja-games/sn/send"
	"go.chromium.org/gae/service/mail"
	"golang.org/x/net/context"
)

func init() {
	gob.RegisterName("*game.awardFavorChipPointsEntry", new(awardFavorChipPointsEntry))
	gob.RegisterName("*game.awardSlanderChipPointsEntry", new(awardSlanderChipPointsEntry))
	gob.RegisterName("*game.announceTHWinnersEntry", new(announceTHWinnersEntry))
}

func (g *Game) startEndGamePhase(ctx context.Context) contest.Contests {
	g.Phase = endGameScoring
	g.awardFavorChipPoints()
	g.awardSlanderChipPoints()

	places := g.determinePlaces(ctx)
	g.setWinners(places[0])
	return contest.GenContests(ctx, places)
}

func toIDS(places []Players) [][]int64 {
	sids := make([][]int64, len(places))
	for i, players := range places {
		for _, p := range players {
			sids[i] = append(sids[i], p.User().ID)
		}
	}
	return sids
}

func (g *Game) awardFavorChipPoints() {
	for _, n := range g.Nationalities() {
		for _, p := range g.chipLeaders(n) {
			p.Score += 2
			e := g.newAwardFavorChipPointsEntryFor(p)
			e.Chip = n
		}
	}
}

func (g *Game) chipLeaders(n nationality) Players {
	max := -1
	var leaders Players
	for _, p := range g.Players() {
		switch chips := p.ChipsFor(n); {
		case chips > max:
			max = chips
			leaders = Players{p}
		case chips == max:
			leaders = append(leaders, p)
		}
	}
	return leaders
}

type awardFavorChipPointsEntry struct {
	*Entry
	Chip nationality
}

func (g *Game) newAwardFavorChipPointsEntryFor(p *Player) (e *awardFavorChipPointsEntry) {
	e = new(awardFavorChipPointsEntry)
	e.Entry = g.newEntryFor(p)
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *awardFavorChipPointsEntry) HTML(ctx context.Context) template.HTML {
	g := gameFrom(ctx)
	return restful.HTML("%s scored 2 points for %s favor chips.", g.NameByPID(e.PlayerID), e.Chip)
}

func (g *Game) awardSlanderChipPoints() {
	for _, p := range g.Players() {
		slanderVP := 0
		for _, chip := range p.SlanderChips {
			if chip {
				slanderVP++
			}
		}
		p.Score += slanderVP
		e := g.newAwardSlanderChipPointsEntryFor(p)
		e.Scored = slanderVP
	}
}

type awardSlanderChipPointsEntry struct {
	*Entry
	Scored int
}

func (g *Game) newAwardSlanderChipPointsEntryFor(p *Player) (e *awardSlanderChipPointsEntry) {
	e = new(awardSlanderChipPointsEntry)
	e.Entry = g.newEntryFor(p)
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *awardSlanderChipPointsEntry) HTML(ctx context.Context) template.HTML {
	g := gameFrom(ctx)
	return restful.HTML("%s scored %v points for unused slander chips.", g.NameByPID(e.PlayerID), e.Scored)
}

func (g *Game) setWinners(rmap contest.ResultsMap) {
	g.Phase = announceWinners
	g.Status = game.Completed

	g.setCurrentPlayers()
	for key := range rmap {
		p := g.playerByUserID(key.IntID())
		g.WinnerIDS = append(g.WinnerIDS, p.ID())
	}

	g.newAnnounceWinnersEntry()
}

type announceTHWinnersEntry struct {
	*Entry
}

func (g *Game) newAnnounceWinnersEntry() (e *announceTHWinnersEntry) {
	e = new(announceTHWinnersEntry)
	e.Entry = g.newEntry()
	g.Log = append(g.Log, e)
	return
}

func (e *announceTHWinnersEntry) HTML(ctx context.Context) template.HTML {
	g := gameFrom(ctx)
	names := make([]string, len(g.Winnerers()))
	for i, winner := range g.Winnerers() {
		names[i] = g.NameFor(winner)
	}
	return restful.HTML("Congratulations: %s.", restful.ToSentence(names))
}

func (g *Game) winners() (ps Players) {
	switch length := len(g.WinnerIDS); length {
	case 0:
	default:
		ps = make(Players, length)
		for i, pid := range g.WinnerIDS {
			player := g.PlayerByID(pid)
			ps[i] = player
		}
	}
	return
}

func (g *Game) sendEndGameNotifications(ctx context.Context) error {
	ms := make([]*mail.Message, len(g.Players()))
	sender := "webmaster@slothninja.com"
	subject := fmt.Sprintf("SlothNinja Games: Tammany Hall #%d Has Ended", g.ID)
	var body string
	body += `!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN">
		<html>
			<head>
				<meta http-equiv="content-type" content="text/html; charset=ISO-8859-1">
			</head>
			<body bgcolor="#ffffff" text="#000000">`
	for _, p := range g.Players() {
		body += fmt.Sprintf("<p>%s scored %d points.</p>", g.NameFor(p), p.Score)
	}

	var names []string
	for _, p := range g.winners() {
		names = append(names, g.NameFor(p))
	}
	body += fmt.Sprintf("<p>Congratulations to: %s.</p>", restful.ToSentence(names))

	body += `
			</body>
		</html>`

	for i, p := range g.Players() {
		ms[i] = &mail.Message{
			To:       []string{p.User().Email},
			Sender:   sender,
			Subject:  subject,
			HTMLBody: body,
		}
	}
	return send.Message(ctx, ms...)
}
