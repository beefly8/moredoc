package model

import (
	// "fmt"
	// "strings"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	CommentStatusPending  = iota // 待审核
	CommentStatusApproved        // 已审核
	CommentStatusRejected        // 已拒绝
)

type Comment struct {
	Id           int64      `form:"id" json:"id,omitempty" gorm:"primaryKey;autoIncrement;column:id;comment:;"`
	UserId       int64      `form:"user_id" json:"user_id,omitempty" gorm:"column:user_id;type:bigint;size:20;index:idx_user_id;comment:发布评论的用户;"`
	ParentId     int64      `form:"parent_id" json:"parent_id,omitempty" gorm:"column:parent_id;type:bigint;size:20;default:0;comment:上级ID;index:idx_parent_id;"`
	Content      string     `form:"content" json:"content,omitempty" gorm:"column:content;type:text;comment:评论内容;"`
	DocumentId   int64      `form:"document_id" json:"document_id,omitempty" gorm:"column:document_id;type:bigint;size:20;default:0;comment:兼容字段，文档ID或文章ID;index:idx_document_id;"`
	Status       int8       `form:"status" json:"status,omitempty" gorm:"column:status;type:integer;size:4;default:0;comment:0 待审，1过审，2拒绝;"`
	CommentCount int        `form:"comment_count" json:"comment_count,omitempty" gorm:"column:comment_count;type:integer;size:11;default:0;comment:评论数量;"`
	IP           string     `form:"ip" json:"ip,omitempty" gorm:"column:ip;type:varchar(64);size:64;default:'';comment:IP地址;"`
	CreatedAt    *time.Time `form:"created_at" json:"created_at,omitempty" gorm:"column:created_at;type:timestamp;comment:评论时间;index:idx_created_at;"`
	UpdatedAt    *time.Time `form:"updated_at" json:"updated_at,omitempty" gorm:"column:updated_at;type:timestamp;comment:评论更新时间;"`
	Type         int32      `form:"type" json:"type,omitempty" gorm:"column:type;type:integer;size:11;default:0;comment:评论类型，0表示文档评论，1表示文章评论;index:idx_type"` // 枚举见CategoryType
}

func (Comment) TableName() string {
	return tablePrefix + "comment"
}

// CreateComment 创建Comment
func (m *DBModel) CreateDocumentComment(comment *Comment) (err error) {
	doc := &Document{}
	m.db.Where("id = ?", comment.DocumentId).Select("id", "title", "user_id", "uuid").Find(doc)

	tx := m.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	err = tx.Create(comment).Error
	if err != nil {
		m.logger.Error("CreateComment", zap.Error(err))
		return
	}

	// 文档评论数+1
	err = tx.Model(&Document{}).Where("id = ?", comment.DocumentId).Update("comment_count", gorm.Expr("comment_count + ?", 1)).Error
	if err != nil {
		m.logger.Error("CreateComment", zap.Error(err))
		return
	}

	// 用户评论数+1
	err = tx.Model(&User{}).Where("id = ?", comment.UserId).Update("comment_count", gorm.Expr("comment_count + ?", 1)).Error
	if err != nil {
		m.logger.Error("CreateComment", zap.Error(err))
		return
	}

	dynamic := &Dynamic{
		UserId: comment.UserId,
		Type:   DynamicTypeComment,
	}
	// 更新上级评论的评论数
	if comment.ParentId > 0 {
		err = tx.Model(&Comment{}).Where("id = ?", comment.ParentId).Update("comment_count", gorm.Expr("comment_count + ?", 1)).Error
		if err != nil {
			m.logger.Error("CreateComment", zap.Error(err))
			return
		}
		dynamic.Content = fmt.Sprintf(`在文档《<a href="/document/%s">%s</a>》中回复了评论`, doc.UUID, html.EscapeString(doc.Title))
	} else {
		dynamic.Content = fmt.Sprintf(`评论了文档《<a href="/document/%s">%s</a>》`, doc.UUID, html.EscapeString(doc.Title))
	}

	// 增加评论动态
	err = tx.Create(dynamic).Error
	if err != nil {
		m.logger.Error("CreateComment", zap.Error(err))
		return
	}

	// 是否可以获得积分奖励
	canRewarded := false
	cfgScore := m.GetConfigOfScore(ConfigScoreDocumentCommented, ConfigScoreDocumentCommentedLimit)
	// 用户文档自评，积分不增加
	if comment.UserId != doc.UserId && cfgScore.DocumentCommented > 0 && cfgScore.DocumentCommentedLimit > 0 {
		var count int64
		errCount := tx.Model(&Comment{}).Where("document_id = ? and created_at > ?", comment.DocumentId, time.Now().Format("2006-01-02")).Count(&count).Error
		if errCount != nil && errCount != gorm.ErrRecordNotFound {
			m.logger.Error("CreateComment", zap.Error(errCount))
			return
		}

		// 用户积分增加
		if int32(count) < cfgScore.DocumentCommentedLimit {
			err = tx.Model(&User{}).Where("id = ?", doc.UserId).Update("credit_count", gorm.Expr("credit_count + ?", cfgScore.DocumentCommented)).Error
			if err != nil {
				m.logger.Error("CreateComment", zap.Error(err))
				return
			}
			canRewarded = true
		}
	}

	// 被评论的文档作者增加动态和积分
	newDynamic := &Dynamic{
		UserId:  doc.UserId,
		Type:    DynamicTypeComment,
		Content: fmt.Sprintf(`您上传的文档《<a href="/document/%s">%s</a>》被评论了`, doc.UUID, html.EscapeString(doc.Title)),
	}
	if canRewarded {
		newDynamic.Content = fmt.Sprintf(`您上传的文档《<a href="/document/%s">%s</a>》被评论了，获得 %d %s奖励`, doc.UUID, html.EscapeString(doc.Title), cfgScore.DocumentCommented, cfgScore.CreditName)
	}

	err = tx.Create(newDynamic).Error
	if err != nil {
		m.logger.Error("CreateComment", zap.Error(err))
		return
	}
	return
}

// 创建文章评论
func (m *DBModel) CreateArticleComment(comment *Comment) (err error) {
	article := &Article{}
	m.db.Where("id = ?", comment.DocumentId).Select("id", "title", "user_id", "identifier").Find(article)
	if article.Id == 0 {
		return errors.New("文章不存在")
	}

	tx := m.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	err = tx.Create(comment).Error
	if err != nil {
		m.logger.Error("CreateComment", zap.Error(err))
		return
	}

	// 文档评论数+1
	err = tx.Model(article).Where("id = ?", comment.DocumentId).Update("comment_count", gorm.Expr("comment_count + ?", 1)).Error
	if err != nil {
		m.logger.Error("CreateComment", zap.Error(err))
		return
	}

	// 用户评论数+1
	err = tx.Model(&User{}).Where("id = ?", comment.UserId).Update("comment_count", gorm.Expr("comment_count + ?", 1)).Error
	if err != nil {
		m.logger.Error("CreateComment", zap.Error(err))
		return
	}

	dynamic := &Dynamic{
		UserId: comment.UserId,
		Type:   DynamicTypeComment,
	}
	// 更新上级评论的评论数
	if comment.ParentId > 0 {
		err = tx.Model(&Comment{}).Where("id = ?", comment.ParentId).Update("comment_count", gorm.Expr("comment_count + ?", 1)).Error
		if err != nil {
			m.logger.Error("CreateComment", zap.Error(err))
			return
		}
		dynamic.Content = fmt.Sprintf(`在文章《<a href="/article/%s">%s</a>》中回复了评论`, article.Identifier, html.EscapeString(article.Title))
	} else {
		dynamic.Content = fmt.Sprintf(`评论了文档《<a href="/article/%s">%s</a>》`, article.Identifier, html.EscapeString(article.Title))
	}

	// 增加评论动态
	err = tx.Create(dynamic).Error
	if err != nil {
		m.logger.Error("CreateComment", zap.Error(err))
		return
	}
	return
}

// UpdateComment 更新Comment，如果需要更新指定字段，则请指定updateFields参数
func (m *DBModel) UpdateComment(comment *Comment, updateFields ...string) (err error) {
	db := m.db.Model(comment)
	tableName := Comment{}.TableName()

	updateFields = m.FilterValidFields(tableName, updateFields...)
	if len(updateFields) > 0 { // 更新指定字段
		db = db.Select(updateFields)
	} else { // 更新全部字段，包括零值字段
		db = db.Select(m.GetTableFields(tableName))
	}

	err = db.Where("id = ?", comment.Id).Updates(comment).Error
	if err != nil {
		m.logger.Error("UpdateComment", zap.Error(err))
	}
	return
}

// GetComment 根据id获取Comment
func (m *DBModel) GetComment(id interface{}, fields ...string) (comment Comment, err error) {
	db := m.db

	fields = m.FilterValidFields(Comment{}.TableName(), fields...)
	if len(fields) > 0 {
		db = db.Select(fields)
	}

	err = db.Where("id = ?", id).First(&comment).Error
	return
}

type OptionGetCommentList struct {
	Page         int
	Size         int
	WithCount    bool                      // 是否返回总数
	Ids          []interface{}             // id列表
	SelectFields []string                  // 查询字段
	QueryRange   map[string][2]interface{} // map[field][]{min,max}
	QueryIn      map[string][]interface{}  // map[field][]{value1,value2,...}
	QueryLike    map[string][]interface{}  // map[field][]{value1,value2,...}
	Sort         []string
}

// GetCommentList 获取Comment列表
func (m *DBModel) GetCommentList(opt *OptionGetCommentList) (commentList []Comment, total int64, err error) {
	tableName := Comment{}.TableName()
	db := m.db.Model(&Comment{})
	db = m.generateQueryRange(db, tableName, opt.QueryRange)
	db = m.generateQueryIn(db, tableName, opt.QueryIn)
	db = m.generateQueryLike(db, tableName, opt.QueryLike)

	if len(opt.Ids) > 0 {
		db = db.Where("id in (?)", opt.Ids)
	}

	if opt.WithCount {
		err = db.Count(&total).Error
		if err != nil {
			m.logger.Error("GetCommentList", zap.Error(err))
			return
		}
	}

	opt.SelectFields = m.FilterValidFields(tableName, opt.SelectFields...)
	if len(opt.SelectFields) > 0 {
		db = db.Select(opt.SelectFields)
	}

	db = m.generateQuerySort(db, tableName, opt.Sort)

	db = db.Offset((opt.Page - 1) * opt.Size).Limit(opt.Size)

	err = db.Find(&commentList).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		m.logger.Error("GetCommentList", zap.Error(err))
	}
	return
}

// DeleteComment 删除数据
// 删除评论之后，对应文档的评论数量也要减少，对应的父级文档评论数量也要减少，用户评论数量也要减少
func (m *DBModel) DeleteComment(ids []int64, limitUserId ...int64) (err error) {
	tx := m.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	var (
		comments []Comment
		user     = &User{}
		document = &Document{}
	)
	cond := []string{"id in (?)"}
	args := []interface{}{ids}
	if len(limitUserId) > 0 {
		cond = append(cond, "user_id in (?)")
		args = append(args, limitUserId)
	}
	condStr := strings.Join(cond, " and ")
	tx.Where(condStr, args...).Select("id", "parent_id", "document_id", "user_id").Find(&comments)
	if len(comments) == 0 {
		err = errors.New("评论不存在或没有权限删除")
		return err
	}

	err = tx.Where(condStr, args...).Delete(&Comment{}).Error
	if err != nil {
		m.logger.Error("DeleteComment", zap.Error(err))
		return
	}

	for _, comment := range comments {
		// 更新文档评论数
		err = tx.Model(document).Where("id = ?", comment.DocumentId).UpdateColumn("comment_count", gorm.Expr("comment_count - ?", 1)).Error
		if err != nil {
			m.logger.Error("DeleteComment", zap.Error(err))
			return
		}

		// 更新父级评论数
		if comment.ParentId > 0 {
			err = tx.Model(&comment).Where("id = ?", comment.ParentId).UpdateColumn("comment_count", gorm.Expr("comment_count - ?", 1)).Error
			if err != nil {
				m.logger.Error("DeleteComment", zap.Error(err))
				return
			}
		}

		// 更新用户评论数
		err = tx.Model(user).Where("id = ?", comment.UserId).UpdateColumn("comment_count", gorm.Expr("comment_count - ?", 1)).Error
		if err != nil {
			m.logger.Error("DeleteComment", zap.Error(err))
			return
		}
	}

	return
}

func (m *DBModel) UpdateCommentStatus(ids []int64, status int32) (err error) {
	err = m.db.Model(&Comment{}).Where("id in (?) and status != ?", ids, status).Update("status", status).Error
	if err != nil {
		m.logger.Error("UpdateCommentStatus", zap.Error(err))
	}
	return
}

func (m *DBModel) CountComment() (count int64, err error) {
	err = m.db.Model(&Comment{}).Count(&count).Error
	if err != nil {
		m.logger.Error("CountComment", zap.Error(err))
	}
	return
}

func (m *DBModel) GetDefaultCommentStatus(userId int64) (status int) {
	// 默认待审核
	status = CommentStatusPending

	var group Group
	// 查询用户所在用户组，是否评论不需要审核
	err := m.db.Select("g.id").Where("ug.user_id = ? and g.enable_comment_approval = ?", userId, false).Table(Group{}.TableName() + " g").Joins(
		"left join " + UserGroup{}.TableName() + " ug on g.id=ug.group_id",
	).Find(&group).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		m.logger.Error("GetDefaultCommentStatus", zap.Error(err))
		return
	}

	m.logger.Debug("GetDefaultCommentStatus", zap.Any("group", group))

	if group.Id > 0 {
		status = CommentStatusApproved
	}
	return
}
