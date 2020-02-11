package tammany

import (
	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
)

const (
	wardKey   = "ward"
	officeKey = "office"
)

func wardFrom(ctx context.Context) (w *Ward) {
	w, _ = ctx.Value(wardKey).(*Ward)
	return
}

func withWard(c *gin.Context, w *Ward) *gin.Context {
	c.Set(wardKey, w)
	return c
}

func officeFrom(ctx context.Context) (o office) {
	o, _ = ctx.Value(officeKey).(office)
	return
}

func withOffice(c *gin.Context, o office) *gin.Context {
	c.Set(officeKey, o)
	return c
}

func (g *Game) selectArea(ctx context.Context) (tmpl string, act game.ActionType, err error) {
	if g.Phase == placeImmigrant {
		g.getWard(ctx)
		tmpl, act = "tammany/place_immigrant_dialog", game.None
	} else if w := g.getWard(ctx); w != nil {
		g.getWard(ctx)
		tmpl, act = "tammany/place_pieces_dialog", game.None
	} else if o := g.getOffice(ctx); o != noOffice {
		tmpl, act = "tammany/assign_office_dialog", game.None
	} else {
		tmpl, act, err = "tammany/flash_notice", game.None, sn.NewVError("Invalid area selected.")
	}
	return
}

func (g *Game) getWard(ctx context.Context) *Ward {
	c := restful.GinFrom(ctx)
	id, ok := toWardID[c.PostForm("area")]
	if !ok {
		return nil
	}
	w := g.wardByID(id)
	withWard(c, w)
	return w
}

func (g *Game) getOffice(ctx context.Context) office {
	c := restful.GinFrom(ctx)
	o, ok := toOffice[c.PostForm("area")]
	if !ok {
		return noOffice
	}
	withOffice(c, o)
	return o
}
