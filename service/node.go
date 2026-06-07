package service

import (
	"encoding/json"

	"github.com/liangshanbo223/github-demo-project/database"
	"github.com/liangshanbo223/github-demo-project/database/model"
	"github.com/liangshanbo223/github-demo-project/util/common"

	"gorm.io/gorm"
)

type NodeService struct{}

func (n *NodeService) GetAll() (*[]model.Node, error) {
	db := database.GetDB()
	var nodes []model.Node
	err := db.Model(model.Node{}).Find(&nodes).Error
	if err != nil {
		return nil, err
	}
	return &nodes, nil
}

func (n *NodeService) Save(tx *gorm.DB, act string, data json.RawMessage) error {
	var err error

	switch act {
	case "new", "edit":
		var node model.Node
		err = json.Unmarshal(data, &node)
		if err != nil {
			return err
		}

		err = tx.Save(&node).Error
		if err != nil {
			return err
		}
	case "del":
		var id uint
		err = json.Unmarshal(data, &id)
		if err != nil {
			// fallback to string name
			var nameStr string
			if err2 := json.Unmarshal(data, &nameStr); err2 == nil {
				err = tx.Where("name = ?", nameStr).Delete(model.Node{}).Error
				return err
			}
			return err
		}
		if id == 0 {
			return common.NewError("不能删除本地主控节点")
		}
		err = tx.Where("id = ?", id).Delete(model.Node{}).Error
		if err != nil {
			return err
		}
	default:
		return common.NewErrorf("unknown action: %s", act)
	}
	return nil
}
