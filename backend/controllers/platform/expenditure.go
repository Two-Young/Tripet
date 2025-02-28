package platform

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"math/big"
	"net/http"
	"os"
	"time"
	"travel-ai/controllers/socket"
	util2 "travel-ai/controllers/util"
	"travel-ai/log"
	"travel-ai/service/database"
	"travel-ai/service/platform"
	"travel-ai/service/platform/database_io"
	"travel-ai/third_party/opencv"
	"travel-ai/third_party/taggun_receipt_ocr"
	"travel-ai/util"
)

func Expenditures(c *gin.Context) {
	uid := c.GetString("uid")

	var query ExpendituresGetRequestDto
	if err := c.ShouldBindQuery(&query); err != nil {
		log.Error(err)
		util2.AbortWithStrJson(c, http.StatusBadRequest, "invalid request query: "+err.Error())
		return
	}

	// check if session exists
	_, err := database_io.GetSession(query.SessionId)
	if err != nil {
		log.Error(err)
		util2.AbortWithStrJson(c, http.StatusBadRequest, "session does not exist")
		return
	}

	// check if user is in session
	yes, err := platform.IsSessionMember(uid, query.SessionId)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if !yes {
		util2.AbortWithStrJson(c, http.StatusBadRequest, "user is not in session")
		return
	}

	expenditureEntities, err := database_io.GetExpendituresBySessionId(query.SessionId)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	expenditures := make(ExpendituresGetResponseDto, 0)
	for _, e := range expenditureEntities {
		expenditures = append(expenditures, ExpendituresGetResponseItem{
			ExpenditureId: e.ExpenditureId,
			Name:          e.Name,
			TotalPrice:    e.TotalPrice,
			CurrencyCode:  e.CurrencyCode,
			Category:      e.Category,
			PayedAt:       e.PayedAt,
			HasReceipt:    e.HasReceipt,
		})
	}

	c.JSON(http.StatusOK, expenditures)
}

func Expenditure(c *gin.Context) {
	uid := c.GetString("uid")

	var query ExpenditureGetRequestDto
	if err := c.ShouldBindQuery(&query); err != nil {
		log.Error(err)
		util2.AbortWithStrJson(c, http.StatusBadRequest, "invalid request query: "+err.Error())
		return
	}

	// check if expenditure exists
	expenditureEntity, err := database_io.GetExpenditure(query.ExpenditureId)
	if err != nil {
		log.Error(err)
		util2.AbortWithStrJson(c, http.StatusBadRequest, "expenditure does not exist")
		return
	}

	// check if user is in session
	yes, err := platform.IsSessionMember(uid, expenditureEntity.SessionId)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if !yes {
		util2.AbortWithStrJson(c, http.StatusBadRequest, "user is not in session")
		return
	}

	// get payers
	payerEntities, err := database_io.GetExpenditurePayers(query.ExpenditureId)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	payers := make([]string, 0)
	for _, payer := range payerEntities {
		payers = append(payers, payer.UserId)
	}

	// get distribution
	distributionEntities, err := database_io.GetExpenditureDistributions(query.ExpenditureId)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	distribution := make([]ExpenditureGetResponseDistributionItem, 0)
	for _, dist := range distributionEntities {
		distribution = append(distribution, ExpenditureGetResponseDistributionItem{
			UserId: dist.UserId,
			Amount: Fraction{
				Numerator:   dist.Numerator,
				Denominator: dist.Denominator,
			},
		})
	}

	// get items
	itemEntities, err := database_io.GetExpenditureItems(query.ExpenditureId)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	items := make([]ExpenditureGetResponseItem, 0)
	for _, item := range itemEntities {
		// get allocations
		allocations, err := database_io.GetExpenditureItemAllocations(item.ExpenditureItemId)
		if err != nil {
			log.Error(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		allocatedUsers := make([]string, 0)
		for _, allocation := range allocations {
			allocatedUsers = append(allocatedUsers, allocation.UserId)
		}

		items = append(items, ExpenditureGetResponseItem{
			Label:       item.Label,
			Price:       item.Price,
			Allocations: allocatedUsers,
		})
	}

	c.JSON(http.StatusOK, ExpenditureGetResponseDto{
		Name:         expenditureEntity.Name,
		TotalPrice:   expenditureEntity.TotalPrice,
		CurrencyCode: expenditureEntity.CurrencyCode,
		Category:     expenditureEntity.Category,
		PayersId:     payers,
		Distribution: distribution,
		Items:        items,
		PayedAt:      expenditureEntity.PayedAt,
	})
}

func CreateExpenditure(c *gin.Context) {
	uid := c.GetString("uid")

	var body ExpenditureCreateRequestDto
	if err := c.ShouldBindJSON(&body); err != nil {
		log.Error(err)
		util2.AbortWithStrJson(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// check if session exists
	_, err := database_io.GetSession(body.SessionId)
	if err != nil {
		log.Error(err)
		util2.AbortWithStrJson(c, http.StatusBadRequest, "session does not exist")
		return
	}

	// check if user is in session
	yes, err := platform.IsSessionMember(uid, body.SessionId)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if !yes {
		util2.AbortWithStrJson(c, http.StatusBadRequest, "user is not in session")
		return
	}

	// validate name
	if len(body.Name) == 0 {
		log.Error("name is empty")
		util2.AbortWithStrJson(c, http.StatusBadRequest, "name is empty")
		return
	}

	// validate total price
	calculatedTotalPrice := big.NewRat(0, 1)
	ratDistributions := make(map[string]*big.Rat)
	for _, dist := range body.Distribution {
		if dist.Amount.Denominator == 0 {
			log.Error("denominator is zero")
			util2.AbortWithStrJson(c, http.StatusBadRequest, "denominator is zero")
			return
		}
		distribution := big.NewRat(dist.Amount.Numerator, dist.Amount.Denominator)
		calculatedTotalPrice.Add(calculatedTotalPrice, distribution)
		// ignore zero
		if distribution.Cmp(big.NewRat(0, 1)) == 0 {
			continue
		}
		ratDistributions[dist.UserId] = distribution
	}
	totalPriceRat := platform.Float64ToRat(*body.TotalPrice)
	if calculatedTotalPrice.Cmp(totalPriceRat) != 0 {
		log.Errorf("total price does not match distribution: (sum) %s != (total) %s from requested (%f)",
			calculatedTotalPrice, totalPriceRat, *body.TotalPrice)
		util2.AbortWithStrJson(c, http.StatusBadRequest, "total price does not match distribution")
		return
	}

	// validate payers
	if len(body.PayersId) == 0 {
		log.Error("no payer specified")
		util2.AbortWithStrJson(c, http.StatusBadRequest, "no payer specified")
		return
	}

	// validate payers each
	for _, payerId := range body.PayersId {
		yes, err := platform.IsSessionMember(payerId, body.SessionId)
		if err != nil {
			log.Error(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if !yes {
			util2.AbortWithStrJson(c, http.StatusBadRequest, "payer is not in session")
			return
		}
	}

	// validate currency code
	yes, err = platform.IsSupportedCurrency(body.CurrencyCode)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if !yes {
		log.Errorf("invalid currency code: %s", body.CurrencyCode)
		util2.AbortWithStrJson(c, http.StatusBadRequest, "invalid currency code")
		return
	}

	// validate category
	if !platform.IsValidExpenditureCategory(body.Category) {
		log.Errorf("invalid category: %s", body.Category)
		util2.AbortWithStrJson(c, http.StatusBadRequest, "invalid category")
		return
	}

	if len(body.Items) > 0 {
		// validate distribution
		calculatedAllocatedPrice := make(map[string]*big.Rat)
		for _, item := range body.Items {
			price := platform.Float64ToRat(*item.Price)
			if len(item.Allocations) == 0 {
				log.Errorf("no allocation specified for item: %s", item.Label)
				util2.AbortWithStrJson(c, http.StatusBadRequest, "no allocation specified for item")
				return
			}
			allocatedUserCnt := big.NewRat(int64(len(item.Allocations)), 1)
			dividedPrice := new(big.Rat)
			dividedPrice.Quo(price, allocatedUserCnt)

			for _, allocatedUid := range item.Allocations {
				// check if user exists
				yes, err := platform.IsSessionMember(allocatedUid, body.SessionId)
				if err != nil {
					log.Error(err)
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}
				if !yes {
					util2.AbortWithStrJson(c, http.StatusBadRequest, "allocated user is not in session")
					return
				}

				allocatedPrice, ok := calculatedAllocatedPrice[allocatedUid]
				if !ok {
					allocatedPrice = new(big.Rat)
				}
				allocatedPrice.Add(allocatedPrice, dividedPrice)
				calculatedAllocatedPrice[allocatedUid] = allocatedPrice
			}
		}

		for userId, dist := range ratDistributions {
			allocatedPrice, ok := calculatedAllocatedPrice[userId]
			if !ok {
				log.Errorf("distribution user not found in items: %s", userId)
				util2.AbortWithStrJson(c, http.StatusBadRequest, "distribution user not found in items")
				return
			}
			if allocatedPrice.Cmp(dist) != 0 {
				log.Errorf("distribution amount does not match items: %s (calculated: %s)", dist, allocatedPrice)
				util2.AbortWithStrJson(c, http.StatusBadRequest, "distribution amount does not match items")
				return
			}
		}
	}

	tx, err := database.DB.BeginTx(c, nil)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	expenditureId := uuid.New().String()

	// find expenditure
	if body.ExpenditureId != nil {
		_, err := database_io.GetExpenditure(*body.ExpenditureId)
		if err != nil {
			log.Error(err)
			c.AbortWithStatus(http.StatusBadRequest)
			util2.AbortWithStrJsonF(c, http.StatusBadRequest, "expenditure does not exist: %s", *body.ExpenditureId)
			return
		}
		expenditureId = *body.ExpenditureId

		// delete expenditure
		if err := database_io.DeleteExpenditureTx(tx, expenditureId); err != nil {
			_ = tx.Rollback()
			log.Error(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	}

	// insert expenditure
	newExpenditure := database.ExpenditureEntity{
		ExpenditureId: expenditureId,
		Name:          body.Name,
		TotalPrice:    *body.TotalPrice,
		CurrencyCode:  body.CurrencyCode,
		Category:      body.Category,
		SessionId:     body.SessionId,
		PayedAt:       time.UnixMilli(body.PayedAt),
	}
	if err := database_io.InsertExpenditureTx(tx, newExpenditure); err != nil {
		_ = tx.Rollback()
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// insert payers
	for _, payerId := range body.PayersId {
		if err := database_io.InsertExpenditurePayerTx(tx, database.ExpenditurePayerEntity{
			ExpenditureId: expenditureId,
			UserId:        payerId,
		}); err != nil {
			_ = tx.Rollback()
			log.Error(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	}

	// insert distribution
	for userId, dist := range ratDistributions {
		if err := database_io.InsertExpenditureDistributionTx(tx, database.ExpenditureDistributionEntity{
			ExpenditureId: expenditureId,
			UserId:        userId,
			Numerator:     dist.Num().Int64(),
			Denominator:   dist.Denom().Int64(),
		}); err != nil {
			_ = tx.Rollback()
			log.Error(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	}

	// insert items
	for _, item := range body.Items {
		itemId := uuid.New().String()
		if err := database_io.InsertExpenditureItemTx(tx, database.ExpenditureItemEntity{
			ExpenditureItemId: itemId,
			Label:             item.Label,
			Price:             *item.Price,
			ExpenditureId:     expenditureId,
		}); err != nil {
			_ = tx.Rollback()
			log.Error(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// insert allocations
		for _, allocatedUid := range item.Allocations {
			if err := database_io.InsertExpenditureItemAllocationTx(tx, database.ExpenditureItemAllocationEntity{
				ExpenditureItemId: itemId,
				UserId:            allocatedUid,
			}); err != nil {
				_ = tx.Rollback()
				log.Error(err)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	socket.SocketManager.Multicast(body.SessionId, uid, socket.EventExpenditureCreated, newExpenditure)
	c.JSON(http.StatusOK, nil)
}

func DeleteExpenditure(c *gin.Context) {
	uid := c.GetString("uid")

	var body ExpenditureDeleteRequestDto
	if err := c.ShouldBindJSON(&body); err != nil {
		log.Error(err)
		util2.AbortWithStrJson(c, http.StatusBadRequest, "invalid request query: "+err.Error())
		return
	}

	// get expenditure
	expenditureEntity, err := database_io.GetExpenditure(body.ExpenditureId)
	if err != nil {
		log.Error(err)
		util2.AbortWithStrJson(c, http.StatusBadRequest, "expenditure does not exist")
		return
	}

	// check if session exists
	sessionEntity, err := database_io.GetSession(expenditureEntity.SessionId)
	if err != nil {
		log.Error(err)
		util2.AbortWithStrJson(c, http.StatusBadRequest, "session does not exist")
		return
	}

	// check if user is in session
	yes, err := platform.IsSessionMember(uid, sessionEntity.SessionId)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if !yes {
		util2.AbortWithStrJson(c, http.StatusBadRequest, "user is not in session")
		return
	}

	tx, err := database.DB.BeginTx(c, nil)
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if err := database_io.DeleteExpenditureTx(tx, body.ExpenditureId); err != nil {
		_ = tx.Rollback()
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	socket.SocketManager.Multicast(sessionEntity.SessionId, uid, socket.EventExpenditureDeleted, body.ExpenditureId)
	c.JSON(http.StatusOK, nil)
}

func UploadReceipt(c *gin.Context) {
	file, _ := c.FormFile("file")
	if file == nil {
		log.Error("file not found")
		util2.AbortWithStrJson(c, http.StatusBadRequest, "no file uploaded")
		return
	}

	// save file as temp file
	dest, _ := util.GenerateTempFilePath()
	if err := c.SaveUploadedFile(file, dest); err != nil {
		log.Error(err)
		util2.AbortWithErrJson(c, http.StatusInternalServerError, err)
		return
	}
	log.Debug("file temporally saved as: " + dest)

	image, err := util.OpenFileAsImage(dest)
	if err != nil {
		log.Error(err)
		util2.AbortWithErrJson(c, http.StatusInternalServerError, err)
		return
	}

	//processedImage, err := cloud_vision.PreprocessImage(image)
	processedImage, err := opencv.CropReceiptSubImage(image)
	if err != nil {
		log.Error(err)
		util2.AbortWithErrJson(c, http.StatusInternalServerError, err)
		return
	}

	// overwrite file
	if err := util.SaveImageFileAsPng(processedImage, dest, true); err != nil {
		log.Error(err)
		util2.AbortWithErrJson(c, http.StatusInternalServerError, err)
		return
	}

	f, err := os.Open(dest)
	if err != nil {
		log.Error(err)
		util2.AbortWithErrJson(c, http.StatusInternalServerError, err)
		return
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Error(err)
			return
		}

		// delete temp file
		if err := os.Remove(dest); err != nil {
			log.Error(err)
			return
		}
		log.Debug("temp file deleted: " + dest)
	}(f)

	taggunResp, err := taggun_receipt_ocr.ParseReceipt(f)
	if err != nil {
		log.Error(err)
		util2.AbortWithErrJson(c, http.StatusInternalServerError, err)
		return
	}

	totalAmount := taggunResp.TotalAmount.Data
	var totalAmountUnit *string // KRW, USD, JPY, ...
	totalAmountConfident := false
	if taggunResp.TotalAmount.ConfidenceLevel >= 0.5 {
		totalAmount = taggunResp.TotalAmount.Data
		totalAmountUnit = &taggunResp.TotalAmount.CurrencyCode
		totalAmountConfident = true
	} else {
		log.Debugf("total amount confidence level is too low: %v", taggunResp.TotalAmount.ConfidenceLevel)
	}

	type Item struct {
		Name   string
		Amount int
		Price  float64
	}
	items := make(map[int]Item)
	calculatedTotalAmount := 0.0
	for _, amountRaw := range taggunResp.Amounts {
		itemRaw, ok := items[amountRaw.Index]
		if !ok {
			itemRaw = Item{
				Amount: 1,
			}
		}
		itemRaw.Name = amountRaw.Text
		itemRaw.Price = amountRaw.Data
		calculatedTotalAmount += itemRaw.Price
		items[amountRaw.Index] = itemRaw
	}
	for _, numberRaw := range taggunResp.Numbers {
		itemRaw, ok := items[numberRaw.Index]
		if !ok {
			itemRaw = Item{}
		}
		itemRaw.Amount = numberRaw.Data
		items[numberRaw.Index] = itemRaw
	}

	if totalAmountConfident {
		if calculatedTotalAmount != totalAmount {
			// add padding item
			items[-1] = Item{
				Name:   "unknown",
				Amount: 1,
				Price:  totalAmount - calculatedTotalAmount,
			}
		}
	}

	//log.Debugf("total amount: %v", totalAmount)
	//log.Debugf("total amount unit: %v", totalAmountUnit)
	if len(items) == 0 {
		log.Debug(taggunResp)
		log.Error("no item found")
		util2.AbortWithStrJson(c, http.StatusBadRequest, "no item found")
		return
	}

	log.Debugf("Found %d items", len(items))
	subItems := make([]ExpenditureReceiptUploadResponseItem, 0)
	for _, item := range items {
		if item.Name == "" && item.Price == 0 {
			continue
		}
		subItems = append(subItems, ExpenditureReceiptUploadResponseItem{
			Label: item.Name,
			Price: item.Price,
		})
	}

	resp := ExpenditureReceiptUploadResponseDto{
		CurrencyCode: totalAmountUnit,
		Items:        subItems,
	}
	c.JSON(http.StatusOK, resp)
}

func Categories(c *gin.Context) {
	c.JSON(http.StatusOK, platform.SupportedCategories)
}

func UseExpenditureRouter(g *gin.RouterGroup) {
	rg := g.Group("/expenditure")
	rg.GET("/list", Expenditures)
	rg.GET("", Expenditure)
	rg.POST("", CreateExpenditure)
	rg.PUT("", CreateExpenditure)
	rg.DELETE("", DeleteExpenditure)

	rg.POST("/receipt", UploadReceipt)

	rg.GET("/categories", Categories)
}
