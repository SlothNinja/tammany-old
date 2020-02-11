package tammany

import (
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/type"
	"go.chromium.org/gae/service/datastore"
	"golang.org/x/net/context"
)

const kind = "Game"

// New creates a new game.
func New(ctx context.Context) (g *Game) {
	g = new(Game)
	g.Header = game.NewHeader(ctx, g)
	g.State = newState()
	g.Parent = pk(ctx)
	g.Type = gType.Tammany
	return
}

func newState() *State {
	return new(State)
}

func pk(ctx context.Context) *datastore.Key {
	return datastore.NewKey(ctx, gType.Tammany.SString(), "root", 0, game.GamesRoot(ctx))
}

func newKey(ctx context.Context, id int64) *datastore.Key {
	return datastore.NewKey(ctx, "Game", "", id, pk(ctx))
}

func (g *Game) init(ctx context.Context) (err error) {
	if err = g.Header.AfterLoad(g); err != nil {
		return
	}

	for _, player := range g.Players() {
		player.init(g)
	}

	return
}

func (g *Game) afterCache(ctx context.Context) error {
	return g.init(ctx)
}
