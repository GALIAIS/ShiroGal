package sync

import (
	"fmt"
	"galgame-gui/internal/api"
	"galgame-gui/internal/database"
	"time"
)

func Run(db *database.Service, sourceRepo *api.SourceRepository) error {

	remoteIDs, err := sourceRepo.GetAllGameIDs()
	if err != nil {
		return fmt.Errorf("从数据源获取所有活跃ID失败: %w", err)
	}

	localIDs, err := db.GetAllGameIDs()
	if err != nil {
		return fmt.Errorf("获取本地所有游戏ID失败: %w", err)
	}

	remoteIDMap := make(map[int64]struct{}, len(remoteIDs))
	for _, id := range remoteIDs {
		remoteIDMap[id] = struct{}{}
	}

	var idsToDelete []int64
	for _, id := range localIDs {
		if _, found := remoteIDMap[id]; !found {
			idsToDelete = append(idsToDelete, id)
		}
	}

	if len(idsToDelete) > 0 {
		_, err := db.DeleteGames(idsToDelete)
		if err != nil {
			return fmt.Errorf("删除本地数据库中的过时数据失败: %w", err)
		}
	}

	latestTime, err := db.GetLatestTimestamp()
	if err != nil {
		// 如果获取时间戳失败（例如数据库为空），我们从一个很早的时间开始同步
		latestTime = time.Time{} // 使用零时，即公元1年1月1日
	}

	updates, err := sourceRepo.GetGamesSince(latestTime)
	if err != nil {
		return fmt.Errorf("从数据源获取更新失败: %w", err)
	}

	if len(updates) == 0 {
		return nil
	}

	_, err = db.UpsertGames(updates)
	if err != nil {
		return fmt.Errorf("更新本地数据库失败: %w", err)
	}

	return nil
}
