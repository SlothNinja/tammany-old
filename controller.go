package tammany

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/codec"
	"bitbucket.org/SlothNinja/slothninja-games/sn/color"
	"bitbucket.org/SlothNinja/slothninja-games/sn/contest"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/mlog"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"bitbucket.org/SlothNinja/slothninja-games/sn/type"
	"bitbucket.org/SlothNinja/slothninja-games/sn/user"
	"bitbucket.org/SlothNinja/slothninja-games/sn/user/stats"
	"github.com/gin-gonic/gin"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/gae/service/info"
	"go.chromium.org/gae/service/memcache"
	"golang.org/x/net/context"
)

const (
	gameKey   = "Game"
	homePath  = "/"
	jsonKey   = "JSON"
	statusKey = "Status"
	hParam    = "hid"
)

func gameFrom(ctx context.Context) (g *Game) {
	g, _ = ctx.Value(gameKey).(*Game)
	return
}

func withGame(c *gin.Context, g *Game) (ret *gin.Context) {
	ret = c
	c.Set(gameKey, g)
	return
}

func jsonFrom(ctx context.Context) (g *Game) {
	g, _ = ctx.Value(jsonKey).(*Game)
	return
}

func withJSON(c *gin.Context, g *Game) (ret *gin.Context) {
	ret = c
	c.Set(jsonKey, g)
	return
}

//type Action func(*restful.Context, *Game, url.Values) (string, game.ActionType, error)
//
//var actionMap = map[string]Action{
//	"select-area":         selectArea,
//	"assign-office":       assignOffice,
//	"place-pieces":        placePieces,
//	"remove":              removeImmigrant,
//	"move-from":           moveFrom,
//	"move-to":             moveTo,
//	"place-lockup-marker": placeLockupMarker,
//	"deputy-take-chip":    deputyTakeChip,
//	"take-chip":           takeChip,
//	"slander":             slander,
//	"bid":                 bid,
//	"undo":                undoAction,
//	"redo":                redoAction,
//	"reset":               resetTurn,
//	"finish":              finishTurn,
//	"game-state":          adminState,
//	"player":              adminPlayer,
//	"ward":                adminWard,
//	"castle-garden":       adminCastleGarden,
//	"immigrant-bag":       adminImmigrantBag,
//	"warn-office":         warnOffice,
//	"confirm-finish":      confirmFinishTurn,
//	"cancel-finish":       cancelFinish,
//}

func (g *Game) update(ctx context.Context) (tmpl string, t game.ActionType, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	switch a := restful.GinFrom(ctx).PostForm("action"); a {
	case "select-area":
		tmpl, t, err = g.selectArea(ctx)
	case "assign-office":
		tmpl, t, err = g.assignOffice(ctx)
	case "place-pieces":
		tmpl, t, err = g.placePieces(ctx)
	case "remove":
		tmpl, t, err = g.removeImmigrant(ctx)
	case "move-from":
		tmpl, t, err = g.moveFrom(ctx)
	case "move-to":
		tmpl, t, err = g.moveTo(ctx)
	case "place-lockup-marker":
		tmpl, t, err = g.placeLockupMarker(ctx)
	case "deputy-take-chip":
		tmpl, t, err = g.deputyTakeChip(ctx)
	case "take-chip":
		tmpl, t, err = g.takeChip(ctx)
	case "slander":
		tmpl, t, err = g.slander(ctx)
	case "bid":
		tmpl, t, err = g.bid(ctx)
	case "undo":
		tmpl, t, err = g.undoAction(ctx)
	case "redo":
		tmpl, t, err = g.redoAction(ctx)
	case "reset":
		tmpl, t, err = g.resetTurn(ctx)
	case "cancel-finish":
		tmpl, t, err = g.cancelFinish(ctx)
	case "game-state":
		tmpl, t, err = g.adminState(ctx)
	case "player":
		tmpl, t, err = g.adminPlayer(ctx)
	case "ward":
		tmpl, t, err = g.adminWard(ctx)
	case "castle-garden":
		tmpl, t, err = g.adminCastleGarden(ctx)
	case "immigrant-bag":
		tmpl, t, err = g.adminImmigrantBag(ctx)
	default:
		tmpl, t, err = "tammany/flash_notice", game.None, sn.NewVError("%v is not a valid action.", a)
	}
	return
}

func show(prefix string) (f gin.HandlerFunc) {
	f = func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")

		g := gameFrom(ctx)
		cu := user.CurrentFrom(ctx)
		c.HTML(http.StatusOK, prefix+"/show", gin.H{
			"Context":    ctx,
			"VersionID":  info.VersionID(ctx),
			"CUser":      cu,
			"Game":       g,
			"IsAdmin":    user.IsAdmin(ctx),
			"Admin":      game.AdminFrom(ctx),
			"MessageLog": mlog.From(ctx),
			"ColorMap":   color.MapFrom(ctx),
			"Notices":    restful.NoticesFrom(ctx),
			"Errors":     restful.ErrorsFrom(ctx),
		})
	}
	return
}

func update(prefix string) (f gin.HandlerFunc) {
	f = func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")

		g := gameFrom(ctx)
		if g == nil {
			log.Errorf(ctx, "Controller#Update Game Not Found")
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}
		template, actionType, err := g.update(ctx)
		switch {
		case err != nil && sn.IsVError(err):
			restful.AddErrorf(ctx, "%v", err)
			withJSON(c, g)
		case err != nil:
			log.Errorf(ctx, err.Error())
			c.Redirect(http.StatusSeeOther, homePath)
			return
		case actionType == game.Cache:
			if err := g.cache(ctx); err != nil {
				restful.AddErrorf(ctx, "%v", err)
			}
			//mkey := g.UndoKey(c)
			//item := memcache.NewItem(ctx, mkey).SetExpiration(time.Minute * 30)
			//v, err := codec.Encode(g)
			//if err != nil {
			//	log.Errorf(c, "Controller#Update Cache Error: %s", err)
			//	c.Redirect(http.StatusSeeOther, showPath(c, prefix))
			//	return
			//}
			//item.SetValue(v)
			//if err := memcache.Set(ctx, item); err != nil {
			//	log.Errorf(c, "Controller#Update Cache Error: %s", err)
			//	c.Redirect(http.StatusSeeOther, showPath(c, prefix))
			//	return
			//}
			//		case actionType == game.SaveAndStatUpdate:
			//			if err := g.saveAndUpdateStats(c); err != nil {
			//				log.Errorf(c, "%s", err)
			//				restful.AddErrorf(c, "Controller#Update SaveAndStatUpdate Error: %s", err)
			//				c.Redirect(http.StatusSeeOther, showPath(c, prefix))
			//				return
			//			}
		case actionType == game.Save:
			if err := g.save(ctx); err != nil {
				log.Errorf(ctx, "%s", err)
				restful.AddErrorf(ctx, "Controller#Update Save Error: %s", err)
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
				return
			}
		case actionType == game.Undo:
			mkey := g.UndoKey(ctx)
			if err := memcache.Delete(ctx, mkey); err != nil && err != memcache.ErrCacheMiss {
				log.Errorf(ctx, "memcache.Delete error: %s", err)
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
				return
			}
		}

		switch jData := jsonFrom(ctx); {
		case jData != nil && template == "json":
			log.Debugf(ctx, "jData: %v", jData)
			log.Debugf(ctx, "template: %v", template)
			c.JSON(http.StatusOK, jData)
		case template == "":
			log.Debugf(ctx, "template: %v", template)
			c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
		default:
			log.Debugf(ctx, "template: %v", template)
			cu := user.CurrentFrom(ctx)

			d := gin.H{
				"Context":   ctx,
				"VersionID": info.VersionID(ctx),
				"CUser":     cu,
				"Game":      g,
				"Ward":      wardFrom(ctx),
				"Office":    officeFrom(ctx),
				"IsAdmin":   user.IsAdmin(ctx),
				"Notices":   restful.NoticesFrom(ctx),
				"Errors":    restful.ErrorsFrom(ctx),
			}
			log.Debugf(ctx, "d: %#v", d)
			c.HTML(http.StatusOK, template, d)
		}
	}
	return
}
func (g *Game) save(ctx context.Context, es ...interface{}) (err error) {
	err = datastore.RunInTransaction(ctx, func(tc context.Context) (terr error) {
		oldG := New(tc)
		if ok := datastore.PopulateKey(oldG.Header, datastore.KeyForObj(tc, g.Header)); !ok {
			terr = fmt.Errorf("unable to populate game with key")
			return
		}

		if terr = datastore.Get(tc, oldG.Header); terr != nil {
			return
		}

		if oldG.UpdatedAt != g.UpdatedAt {
			terr = fmt.Errorf("game state changed unexpectantly")
			return
		}

		if terr = g.encode(ctx); terr != nil {
			return
		}

		if terr = datastore.Put(tc, append(es, g.Header)); terr != nil {
			return
		}

		if terr = memcache.Delete(tc, g.UndoKey(tc)); terr == memcache.ErrCacheMiss {
			terr = nil
		}
		return
	}, &datastore.TransactionOptions{XG: true})
	return
}

func (g *Game) encode(ctx context.Context) (err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	var encoded []byte
	if encoded, err = codec.Encode(g.State); err != nil {
		return
	}
	g.SavedState = encoded
	g.updateHeader()

	return
}

func (g *Game) cache(ctx context.Context) error {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	item := memcache.NewItem(ctx, g.UndoKey(ctx)).SetExpiration(time.Minute * 30)
	v, err := codec.Encode(g)
	if err != nil {
		return err
	}
	item.SetValue(v)
	return memcache.Set(ctx, item)
}

func wrap(s *stats.Stats, cs contest.Contests) (es []interface{}) {
	es = make([]interface{}, len(cs)+1)
	es[0] = s
	for i, c := range cs {
		es[i+1] = c
	}
	return
}

func showPath(prefix, hid string) string {
	return fmt.Sprintf("/%s/game/show/%s", prefix, hid)
}

func recruitingPath(prefix string) string {
	return fmt.Sprintf("/%s/games/recruiting", prefix)
}

func newPath(prefix string) string {
	return fmt.Sprintf("/%s/game/new", prefix)
}

func newGamer(ctx context.Context) game.Gamer {
	return New(ctx)
}

func undo(prefix string) (f gin.HandlerFunc) {
	f = func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")
		c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))

		g := gameFrom(ctx)
		if g == nil {
			log.Errorf(ctx, "Controller#Update Game Not Found")
			return
		}
		mkey := g.UndoKey(ctx)
		if err := memcache.Delete(ctx, mkey); err != nil && err != memcache.ErrCacheMiss {
			log.Errorf(ctx, "Controller#Undo Error: %s", err)
		}
	}
	return
}

func index(prefix string) (f gin.HandlerFunc) {
	f = func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")

		gs := game.GamersFrom(ctx)
		switch status := game.StatusFrom(ctx); status {
		case game.Recruiting:
			c.HTML(http.StatusOK, "shared/invitation_index", gin.H{
				"Context":   ctx,
				"VersionID": info.VersionID(ctx),
				"CUser":     user.CurrentFrom(ctx),
				"Games":     gs,
				"Type":      gType.Indonesia.String(),
			})
		default:
			c.HTML(http.StatusOK, "shared/games_index", gin.H{
				"Context":   ctx,
				"VersionID": info.VersionID(ctx),
				"CUser":     user.CurrentFrom(ctx),
				"Games":     gs,
				"Type":      gType.Indonesia.String(),
				"Status":    status,
			})
		}
	}
	return
}

func newAction(prefix string) (f gin.HandlerFunc) {
	f = func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")

		g := New(ctx)
		withGame(c, g)
		if err := g.FromParams(ctx, gType.GOT); err != nil {
			log.Errorf(ctx, err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		c.HTML(http.StatusOK, prefix+"/new", gin.H{
			"Context":   ctx,
			"VersionID": info.VersionID(ctx),
			"CUser":     user.CurrentFrom(ctx),
			"Game":      g,
		})
	}
	return
}

func create(prefix string) (f gin.HandlerFunc) {
	f = func(c *gin.Context) {
		ctx := restful.ContextFrom(c)

		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")
		defer c.Redirect(http.StatusSeeOther, recruitingPath(prefix))

		g := New(ctx)
		withGame(c, g)

		var err error
		if err = g.FromParams(ctx, g.Type); err == nil {
			err = g.encode(ctx)
		}

		if err == nil {
			err = datastore.RunInTransaction(ctx, func(tc context.Context) (err error) {
				if err = datastore.Put(tc, g.Header); err != nil {
					return
				}

				m := mlog.New()
				m.ID = g.ID
				return datastore.Put(tc, m)

			}, &datastore.TransactionOptions{XG: true})
		}

		if err == nil {
			restful.AddNoticef(ctx, "<div>%s created.</div>", g.Title)
		} else {
			log.Errorf(ctx, err.Error())
		}
	}
	return
}

func accept(prefix string) (f gin.HandlerFunc) {
	f = func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")
		defer c.Redirect(http.StatusSeeOther, recruitingPath(prefix))

		g := gameFrom(ctx)
		if g == nil {
			log.Errorf(ctx, "game not found")
			return
		}

		var (
			start bool
			err   error
		)

		u := user.CurrentFrom(ctx)
		if start, err = g.Accept(ctx, u); err == nil && start {
			g.start(ctx)
		}

		if err == nil {
			err = g.save(ctx)
		}

		if err == nil && start {
			g.SendTurnNotificationsTo(ctx, g.CurrentPlayer())
		}

		if err != nil {
			log.Errorf(ctx, err.Error())
		}

	}
	return
}

func drop(prefix string) (f gin.HandlerFunc) {
	f = func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")
		defer c.Redirect(http.StatusSeeOther, recruitingPath(prefix))

		g := gameFrom(ctx)
		if g == nil {
			log.Errorf(ctx, "game not found")
			return
		}

		var err error

		u := user.CurrentFrom(ctx)
		if err = g.Drop(u); err == nil {
			err = g.save(ctx)
		}

		if err != nil {
			log.Errorf(ctx, err.Error())
			restful.AddErrorf(ctx, err.Error())
		}

	}
	return
}

func fetch(c *gin.Context) {
	ctx := restful.ContextFrom(c)
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	// create Gamer
	id, err := strconv.ParseInt(c.Param("hid"), 10, 64)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	g := New(ctx)
	g.ID = id

	switch action := c.PostForm("action"); {
	case action == "reset":
		// pull from memcache/datastore
		// same as undo & !MultiUndo
		fallthrough
	case action == "undo":
		// pull from memcache/datastore
		if err := dsGet(ctx, g); err != nil {
			c.Redirect(http.StatusSeeOther, homePath)
		}
	default:
		if user.CurrentFrom(ctx) != nil {
			// pull from memcache and return if successful; otherwise pull from datastore
			if err := mcGet(ctx, g); err == nil {
				return
			}
		}

		if err := dsGet(ctx, g); err != nil {
			c.Redirect(http.StatusSeeOther, homePath)
		}
	}
}

// pull temporary game state from memcache.  Note may be different from value stored in datastore.
func mcGet(ctx context.Context, g *Game) (err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	mkey := g.GetHeader().UndoKey(ctx)

	var item memcache.Item
	if item, err = memcache.GetKey(ctx, mkey); err != nil {
		return
	}

	if err = codec.Decode(g, item.Value()); err != nil {
		return
	}

	if err = g.afterCache(ctx); err == nil {
		color.WithMap(withGame(restful.GinFrom(ctx), g), g.ColorMapFor(user.CurrentFrom(ctx)))
	}
	return
}

// pull game state from memcache/datastore.  returned memcache should be same as datastore.
func dsGet(ctx context.Context, g *Game) (err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	if err = datastore.Get(ctx, g.Header); err == nil && g == nil {
		err = fmt.Errorf("Unable to get game for id: %v", g.ID)
	}

	if err != nil {
		restful.AddErrorf(ctx, err.Error())
		return
	}

	s := newState()
	if err = codec.Decode(&s, g.SavedState); err != nil {
		restful.AddErrorf(ctx, err.Error())
		return
	}
	g.State = s

	if err = g.init(ctx); err != nil {
		restful.AddErrorf(ctx, err.Error())
	} else {
		cm := g.ColorMapFor(user.CurrentFrom(ctx))
		color.WithMap(withGame(restful.GinFrom(ctx), g), cm)
	}
	return
}

func jsonIndexAction(prefix string) (f gin.HandlerFunc) {
	f = func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")

		game.JSONIndexAction(c)
	}
	return
}

func (g *Game) updateHeader() {
	switch g.Phase {
	case gameOver:
		g.Progress = g.PhaseName()
	default:
		g.Progress = fmt.Sprintf("<div>Year: %d</div><div>Phase: %s</div>", g.Year(), g.PhaseName())
	}

	if u := g.Creator; u != nil {
		g.CreatorSID = user.GenID(u.GoogleID)
		g.CreatorName = u.Name
	}

	if l := len(g.Users); l > 0 {
		g.UserSIDS = make([]string, l)
		g.UserNames = make([]string, l)
		g.UserEmails = make([]string, l)
		for i, u := range g.Users {
			g.UserSIDS[i] = user.GenID(u.GoogleID)
			g.UserNames[i] = u.Name
			g.UserEmails[i] = u.Email
		}
	}
}
