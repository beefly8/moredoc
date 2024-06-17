package model

import (
	"time"

	"go.uber.org/zap"
)

type GroupPermission struct {
	Id           int64      `form:"id" json:"id,omitempty" gorm:"primaryKey;autoIncrement;column:id;comment:自增主键;"`
	GroupId      int64      `form:"group_id" json:"group_id,omitempty" gorm:"index:group_permission,unique;index:group_id;column:group_id;comment:组ID;"`
	PermissionId int64      `form:"permission_id" json:"permission_id,omitempty" gorm:"index:group_permission,unique;column:permission_id;comment:权限ID;"`
	CreatedAt    *time.Time `form:"created_at" json:"created_at,omitempty" gorm:"column:created_at;type:timestamp;comment:创建时间;"`
	UpdatedAt    *time.Time `form:"updated_at" json:"updated_at,omitempty" gorm:"column:updated_at;type:timestamp;comment:更新时间;"`
}

func (GroupPermission) TableName() string {
	return tablePrefix + "group_permission"
}

// GetGroupPermissinsByGroupId 根据用户组ID获取用户组权限
func (m *DBModel) GetGroupPermissinsByGroupId(groupId int64) (groupPermissions []*GroupPermission, err error) {
	err = m.db.Where("group_id = ?", groupId).Find(&groupPermissions).Error
	return
}

// 设置权限
func (m *DBModel) UpdateGroupPermissions(groupdId int64, permissionIds []int64) (err error) {
	sess := m.db.Begin()
	defer func() {
		if err != nil {
			sess.Rollback()
		} else {
			sess.Commit()
		}
	}()

	// 删除旧的权限
	err = sess.Where("group_id = ?", groupdId).Delete(&GroupPermission{}).Error
	if err != nil {
		m.logger.Error("delete old permission", zap.Error(err))
		return
	}

	// 添加新的权限
	var (
		permissions     []GroupPermission
		existPermission = make(map[int64]struct{})
	)

	// 去重
	for _, permissionId := range permissionIds {
		if _, ok := existPermission[permissionId]; !ok && permissionId > 0 {
			existPermission[permissionId] = struct{}{}
			permissions = append(permissions, GroupPermission{GroupId: groupdId, PermissionId: permissionId})
		}
	}

	if len(permissions) == 0 {
		return
	}

	err = sess.Create(&permissions).Error
	if err != nil {
		m.logger.Error("create group permission", zap.Error(err))
		return
	}

	return
}
