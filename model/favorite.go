package model

import (
	"errors"
	"fmt"
	pb "moredoc/api/v1"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	FavoritetypeDocument = 0
	FavoritetypeArticle  = 1
)

type Favorite struct {
	Id         int64     `form:"id" json:"id,omitempty" gorm:"primaryKey;autoIncrement;column:id;comment:自增主键;"`
	UserId     int64     `form:"user_id" json:"user_id,omitempty" gorm:"column:user_id;type:bigint;size:20;default:0;comment:用户id;index:idx_user_id;index:idx_user_document_type,unique"`
	DocumentId int64     `form:"document_id" json:"document_id,omitempty" gorm:"column:document_id;type:bigint;size:20;default:0;comment:字段兼容，表示文档ID或者文章ID等;index:idx_user_document_type,unique"`
	Type       int32     `form:"type" json:"type,omitempty" gorm:"column:type;type:integer;size:11;default:0;comment:收藏类型，0表示文档，1表示文章;index:idx_type;index:idx_user_document_type,unique"` // 枚举见 CategoryType
	CreatedAt  time.Time `form:"created_at" json:"created_at,omitempty" gorm:"column:created_at;type:timestamp;comment:;"`
	UpdatedAt  time.Time `form:"updated_at" json:"updated_at,omitempty" gorm:"column:updated_at;type:timestamp;comment:;"`
	IP         string    `form:"ip" json:"ip,omitempty" gorm:"column:ip;type:varchar(64);size:64;default:'';comment:IP地址;"`
}

func (Favorite) TableName() string {
	return tablePrefix + "favorite"
}

// CreateFavorite 创建Favorite
func (m *DBModel) CreateDocumentFavorite(favorite *Favorite) (err error) {
	tx := m.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	// 添加收藏
	err = tx.Create(favorite).Error
	if err != nil {
		m.logger.Error("CreateFavorite", zap.Error(err))
		return
	}

	// 文档收藏数量增加
	err = tx.Model(&Document{}).Where("id = ?", favorite.DocumentId).Update("favorite_count", gorm.Expr("favorite_count + ?", 1)).Error
	if err != nil {
		m.logger.Error("CreateFavorite", zap.Error(err))
		return
	}

	// 用户收藏数量增加
	err = tx.Model(&User{}).Where("id = ?", favorite.UserId).Update("favorite_count", gorm.Expr("favorite_count + ?", 1)).Error
	if err != nil {
		m.logger.Error("CreateFavorite", zap.Error(err))
		return
	}

	doc, _ := m.GetDocument(favorite.DocumentId, "id", "user_id", "title", "uuid")
	if doc.Id == 0 {
		return
	}

	err = tx.Create(&Dynamic{
		UserId:  favorite.UserId,
		Type:    DynamicTypeFavorite,
		Content: fmt.Sprintf(`您收藏了文档《<a href="/document/%s">%s</a>》`, doc.UUID, doc.Title),
	}).Error
	if err != nil {
		m.logger.Error("CreateFavorite", zap.Error(err))
		return
	}

	cfgScore := m.GetConfigOfScore(ConfigScoreDocumentCollected, ConfigScoreDocumentCollectedLimit, ConfigScoreCreditName)
	if doc.UserId != favorite.UserId && cfgScore.DocumentCollected > 0 && cfgScore.DocumentCollectedLimit > 0 {
		var todayFavoriteCount int64
		tx.Model(&Favorite{}).Where("user_id = ? and created_at >= ?", favorite.UserId, time.Now().Format("2006-01-02")).Count(&todayFavoriteCount)
		if int32(todayFavoriteCount) >= cfgScore.DocumentCollectedLimit {
			return
		}

		// 文档分享者积分增加
		err = tx.Model(&User{}).Where("id = ?", doc.UserId).Update("credit_count", gorm.Expr("credit_count + ?", cfgScore.DocumentCollected)).Error
		if err != nil {
			m.logger.Error("CreateFavorite", zap.Error(err))
			return
		}

		err = tx.Create(&Dynamic{
			UserId:  doc.UserId,
			Type:    DynamicTypeFavorite,
			Content: fmt.Sprintf(`您分享的文档《<a href="/document/%s">%s</a>》被收藏，获得 %d %s奖励`, doc.UUID, doc.Title, cfgScore.DocumentCollected, cfgScore.CreditName),
		}).Error
		if err != nil {
			m.logger.Error("CreateFavorite", zap.Error(err))
			return
		}
	}

	return
}

func (m *DBModel) CreateArticleFavorite(favorite *Favorite) (err error) {
	tx := m.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	// 添加收藏
	err = tx.Create(favorite).Error
	if err != nil {
		m.logger.Error("CreateFavorite", zap.Error(err))
		return
	}

	// 文章收藏数量增加
	err = tx.Model(&Article{}).Where("id = ?", favorite.DocumentId).Update("favorite_count", gorm.Expr("favorite_count + ?", 1)).Error
	if err != nil {
		m.logger.Error("CreateFavorite", zap.Error(err))
		return
	}

	// 用户收藏数量增加
	err = tx.Model(&User{}).Where("id = ?", favorite.UserId).Update("favorite_count", gorm.Expr("favorite_count + ?", 1)).Error
	if err != nil {
		m.logger.Error("CreateFavorite", zap.Error(err))
		return
	}

	article, _ := m.GetArticle(favorite.DocumentId, "id", "user_id", "title", "identifier")
	if article.Id == 0 {
		err = errors.New("article not found")
		return
	}

	err = tx.Create(&Dynamic{
		UserId:  favorite.UserId,
		Type:    DynamicTypeFavorite,
		Content: fmt.Sprintf(`您收藏了文章《<a href="/article/%s">%s</a>》`, article.Identifier, article.Title),
	}).Error
	if err != nil {
		m.logger.Error("CreateFavorite", zap.Error(err))
	}
	return
}

func (m *DBModel) GetUserFavorite(userId int64, DocumentId int64, favoriteType int32) (favorite Favorite, err error) {
	err = m.db.Where("user_id = ? and document_id = ? and `type` = ?", userId, DocumentId, favoriteType).First(&favorite).Error
	return
}

// GetFavorite 根据id获取Favorite
func (m *DBModel) GetFavorite(id int64, fields ...string) (favorite Favorite, err error) {
	db := m.db

	fields = m.FilterValidFields(Favorite{}.TableName(), fields...)
	if len(fields) > 0 {
		db = db.Select(fields)
	}

	err = db.Where("id = ?", id).First(&favorite).Error
	return
}

type OptionGetFavoriteList struct {
	Page         int
	Size         int
	WithCount    bool
	SelectFields []string                 // 查询字段
	QueryIn      map[string][]interface{} // map[field][]{value1,value2,...}
}

// GetFavoriteList 获取Favorite列表
func (m *DBModel) GetDocumentFavoriteList(opt *OptionGetFavoriteList, documentStatus ...int) (favoriteList []*pb.Favorite, total int64, err error) {
	tableFavorite := Favorite{}.TableName() + " f"
	db := m.db.
		Table(tableFavorite).
		Joins(
			fmt.Sprintf("left join %s d on f.document_id = d.id", Document{}.TableName()),
		)
	db = m.generateQueryIn(db, tableFavorite, opt.QueryIn)
	db = db.Where("f.type = ?", FavoritetypeDocument)

	if len(documentStatus) > 0 {
		db = db.Where("a.status in (?)", documentStatus)
	}

	if opt.WithCount {
		err = db.Count(&total).Error
		if err != nil {
			m.logger.Error("GetFavoriteList", zap.Error(err))
			return
		}
	}

	opt.SelectFields = m.FilterValidFields(tableFavorite, opt.SelectFields...)
	if len(opt.SelectFields) > 0 {
		db = db.Select(opt.SelectFields)
	}

	db = db.Order("id desc").Offset((opt.Page - 1) * opt.Size).Limit(opt.Size)
	// 注意：size字段要用size_ 才能映射到pb.Favorite
	err = db.Select("f.*, d.title, d.ext, d.score, d.pages, d.size as size_, d.uuid as document_uuid").Find(&favoriteList).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		m.logger.Error("GetFavoriteList", zap.Error(err))
	}
	return
}

// 获取文章收藏列表
func (m *DBModel) GetArticleFavoriteList(opt *OptionGetFavoriteList, articleStatus ...int) (favoriteList []*pb.Favorite, total int64, err error) {
	tableFavorite := Favorite{}.TableName() + " f"
	db := m.db.
		Table(tableFavorite).
		Joins(
			fmt.Sprintf("left join %s a on f.document_id = a.id", Article{}.TableName()),
		)
	db = m.generateQueryIn(db, tableFavorite, opt.QueryIn)

	db = db.Where("f.type = ?", FavoritetypeArticle)
	if len(articleStatus) > 0 {
		db = db.Where("a.status in (?)", articleStatus)
	}

	if opt.WithCount {
		err = db.Count(&total).Error
		if err != nil {
			m.logger.Error("GetFavoriteList", zap.Error(err))
			return
		}
	}

	opt.SelectFields = m.FilterValidFields(tableFavorite, opt.SelectFields...)
	if len(opt.SelectFields) > 0 {
		db = db.Select(opt.SelectFields)
	}

	db = db.Order("id desc").Offset((opt.Page - 1) * opt.Size).Limit(opt.Size)
	// 注意：size字段要用size_ 才能映射到pb.Favorite
	err = db.Select("f.*, a.title, a.identifier as document_uuid").Find(&favoriteList).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		m.logger.Error("GetFavoriteList", zap.Error(err))
	}
	return
}

// DeleteFavorite 取消收藏
func (m *DBModel) DeleteFavorite(userId int64, ids []int64) (err error) {
	if len(ids) == 0 || userId == 0 {
		return
	}

	var favorites []Favorite
	m.db.Where("id in (?) and user_id = ?", ids, userId).Select("id", "user_id", "document_id", "type").Find(&favorites)
	if len(favorites) == 0 {
		return
	}

	tx := m.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	// 删除收藏
	document := &Document{}
	article := &Article{}
	user := &User{}
	for _, favorite := range favorites {
		err = tx.Delete(&favorite).Error
		if err != nil {
			m.logger.Error("DeleteFavorite", zap.Error(err))
			return
		}

		if favorite.Type == 1 {
			// 文章收藏数量减少
			err = tx.Model(article).Where("id = ?", favorite.DocumentId).Update("favorite_count", gorm.Expr("favorite_count - ?", 1)).Error
			if err != nil {
				m.logger.Error("DeleteFavorite", zap.Error(err))
				return
			}
		} else {
			// 文档收藏数量减少
			err = tx.Model(document).Where("id = ?", favorite.DocumentId).Update("favorite_count", gorm.Expr("favorite_count - ?", 1)).Error
			if err != nil {
				m.logger.Error("DeleteFavorite", zap.Error(err))
				return
			}
		}
		// 用户收藏数量减少
		err = tx.Model(user).Where("id = ?", favorite.UserId).Update("favorite_count", gorm.Expr("favorite_count - ?", 1)).Error
		if err != nil {
			m.logger.Error("DeleteFavorite", zap.Error(err))
			return
		}
	}

	return
}
