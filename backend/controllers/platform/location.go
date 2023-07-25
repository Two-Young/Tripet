package platform

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"travel-ai/controllers/util"
	"travel-ai/log"
	"travel-ai/service/database"
	"travel-ai/service/platform"
)

func Locations(c *gin.Context) {
	rawUid, ok := c.Get("uid")
	if !ok {
		log.Error("uid not found")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	uid := rawUid.(string)

	var query locationsRequestDto
	if err := c.ShouldBindQuery(&query); err != nil {
		log.Error(err)
		util.AbortWithStrJson(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// check if user has permission to view locations
	yes, err := platform.DidParticipateInSession(uid, query.SessionId)
	if err != nil {
		log.Error(err)
		util.AbortWithErrJson(c, http.StatusInternalServerError, err)
		return
	}
	if !yes {
		util.AbortWithStrJson(c, http.StatusForbidden, "permission denied")
		return
	}

	// get locations
	var locations []database.LocationEntity
	if err := database.DB.Select(
		&locations,
		"SELECT lid, place_id, name, latitude, longitude, address FROM locations WHERE sid = ?;",
		query.SessionId,
	); err != nil {
		log.Error(err)
		util.AbortWithErrJson(c, http.StatusInternalServerError, err)
		return
	}

	locationResp := make(locationsResponseDto, 0)
	for _, l := range locations {
		locationResp = append(locationResp, locationsResponseItem{
			LocationId: *l.LocationId,
			PlaceId:    *l.PlaceId,
			Name:       *l.Name,
			Latitude:   *l.Latitude,
			Longitude:  *l.Longitude,
			Address:    *l.Address,
		})
	}

	c.JSON(http.StatusOK, locationResp)
}

func CreateLocation(c *gin.Context) {
	rawUid, ok := c.Get("uid")
	if !ok {
		log.Error("uid not found")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	uid := rawUid.(string)

	var body locationCreateRequestDto
	if err := c.ShouldBindJSON(&body); err != nil {
		log.Error(err)
		util.AbortWithStrJson(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// check if user has permission to create location
	yes, err := platform.DidParticipateInSession(uid, body.SessionId)
	if err != nil {
		log.Error(err)
		util.AbortWithErrJson(c, http.StatusInternalServerError, err)
		return
	}
	if !yes {
		util.AbortWithStrJson(c, http.StatusForbidden, "permission denied")
		return
	}

	// create location entity
	locationId := uuid.New().String()
	if _, err := database.DB.Exec(
		"INSERT INTO locations (lid, place_id, name, latitude, longitude, address, sid) VALUES (?, ?, ?, ?, ?, ?, ?);",
		locationId, body.PlaceId, body.Name, body.Latitude, body.Longitude, body.Address, body.SessionId,
	); err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// return location id
	c.JSON(http.StatusOK, locationId)
}

func DeleteLocation(c *gin.Context) {
	rawUid, ok := c.Get("uid")
	if !ok {
		log.Error("uid not found")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	uid := rawUid.(string)

	var body locationDeleteRequestDto
	if err := c.ShouldBindJSON(&body); err != nil {
		log.Error(err)
		util.AbortWithStrJson(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// find session id by location id
	sessionId, err := platform.FindSessionIdByLocationId(body.LocationId)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// check if user has permission to create location
	yes, err := platform.DidParticipateInSession(uid, sessionId)
	if err != nil {
		log.Error(err)
		util.AbortWithErrJson(c, http.StatusInternalServerError, err)
		return
	}
	if !yes {
		util.AbortWithStrJson(c, http.StatusForbidden, "permission denied")
		return
	}

	// delete location entity
	if _, err := database.DB.Exec(
		"DELETE FROM locations WHERE lid = ?;",
		body.LocationId,
	); err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)
}

func UseLocationRouter(g *gin.RouterGroup) {
	rg := g.Group("/location")
	rg.GET("", Locations)
	rg.PUT("", CreateLocation)
	rg.DELETE("", DeleteLocation)
}
