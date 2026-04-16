// server/internal/service/genre.go
package service

import (
	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
	"log"
)

// GenreService 赛道业务逻辑层
type GenreService struct {
	genreDAO *dao.GenreDAO
}

// NewGenreService 创建 GenreService 实例
func NewGenreService() *GenreService {
	return &GenreService{genreDAO: dao.NewGenreDAO()}
}

// SeedDefaults 初始化默认赛道数据（幂等），表中已有数据则跳过
func (s *GenreService) SeedDefaults() {
	count, err := s.genreDAO.Count()
	if err != nil {
		log.Printf("[genre] check genre count failed: %v", err)
		return
	}
	if count > 0 {
		return
	}

	// ---- 第一层：顶级赛道 ----
	topGenres := []model.Genre{
		{Name: "玄幻", Slug: "xuanhuan", Icon: "⚔️", SortOrder: 1},
		{Name: "都市", Slug: "dushi", Icon: "🏙️", SortOrder: 2},
		{Name: "仙侠", Slug: "xianxia", Icon: "🌙", SortOrder: 3},
		{Name: "科幻", Slug: "scifi", Icon: "🚀", SortOrder: 4},
		{Name: "历史", Slug: "lishi", Icon: "📜", SortOrder: 5},
		{Name: "游戏", Slug: "game", Icon: "🎮", SortOrder: 6},
		{Name: "悬疑", Slug: "xuanyi", Icon: "🔍", SortOrder: 7},
		{Name: "奇幻", Slug: "qihuan", Icon: "🐉", SortOrder: 8},
		{Name: "武侠", Slug: "wuxia", Icon: "🥋", SortOrder: 9},
		{Name: "言情", Slug: "yanqing", Icon: "💕", SortOrder: 10},
		{Name: "轻小说", Slug: "lightnovel", Icon: "📖", SortOrder: 11},
		{Name: "短篇", Slug: "duanpian", Icon: "✏️", SortOrder: 12},
	}
	if err := s.genreDAO.SeedGenres(topGenres); err != nil {
		log.Printf("[genre] seed top genres failed: %v", err)
		return
	}

	// ---- 第二层：子赛道 ----
	subGenres := map[string][]model.Genre{
		"xuanhuan": {
			{Name: "东方玄幻", Slug: "xuanhuan-east", Icon: "🏯", SortOrder: 1},
			{Name: "异世大陆", Slug: "xuanhuan-isekai", Icon: "🌍", SortOrder: 2},
			{Name: "王朝争霸", Slug: "xuanhuan-dynasty", Icon: "👑", SortOrder: 3},
			{Name: "高武世界", Slug: "xuanhuan-highwu", Icon: "💪", SortOrder: 4},
		},
		"dushi": {
			{Name: "都市异能", Slug: "dushi-yineng", Icon: "⚡", SortOrder: 1},
			{Name: "都市生活", Slug: "dushi-life", Icon: "☕", SortOrder: 2},
			{Name: "商战职场", Slug: "dushi-biz", Icon: "💼", SortOrder: 3},
			{Name: "娱乐明星", Slug: "dushi-star", Icon: "🌟", SortOrder: 4},
		},
		"xianxia": {
			{Name: "修真文明", Slug: "xianxia-xiuzhen", Icon: "🧘", SortOrder: 1},
			{Name: "幻想修仙", Slug: "xianxia-fantasy", Icon: "☁️", SortOrder: 2},
			{Name: "现代修真", Slug: "xianxia-modern", Icon: "📱", SortOrder: 3},
		},
		"scifi": {
			{Name: "星际文明", Slug: "scifi-space", Icon: "🛸", SortOrder: 1},
			{Name: "超级科技", Slug: "scifi-tech", Icon: "🤖", SortOrder: 2},
			{Name: "末世危机", Slug: "scifi-apocalypse", Icon: "☢️", SortOrder: 3},
			{Name: "时空穿梭", Slug: "scifi-timetravel", Icon: "⏳", SortOrder: 4},
		},
		"lishi": {
			{Name: "架空历史", Slug: "lishi-alt", Icon: "🗺️", SortOrder: 1},
			{Name: "秦汉三国", Slug: "lishi-sanguo", Icon: "🏹", SortOrder: 2},
			{Name: "两宋元明", Slug: "lishi-songming", Icon: "🎎", SortOrder: 3},
			{Name: "民国抗战", Slug: "lishi-minguo", Icon: "🎖️", SortOrder: 4},
		},
		"game": {
			{Name: "电竞网游", Slug: "game-esport", Icon: "🖥️", SortOrder: 1},
			{Name: "虚拟现实", Slug: "game-vr", Icon: "🥽", SortOrder: 2},
			{Name: "游戏异界", Slug: "game-isekai", Icon: "🎲", SortOrder: 3},
		},
		"xuanyi": {
			{Name: "诡秘悬疑", Slug: "xuanyi-horror", Icon: "👻", SortOrder: 1},
			{Name: "探案推理", Slug: "xuanyi-detective", Icon: "🕵️", SortOrder: 2},
			{Name: "古今传奇", Slug: "xuanyi-legend", Icon: "📿", SortOrder: 3},
		},
		"yanqing": {
			{Name: "古代言情", Slug: "yanqing-ancient", Icon: "🏮", SortOrder: 1},
			{Name: "现代言情", Slug: "yanqing-modern", Icon: "💐", SortOrder: 2},
			{Name: "幻想言情", Slug: "yanqing-fantasy", Icon: "🦋", SortOrder: 3},
			{Name: "豪门总裁", Slug: "yanqing-ceo", Icon: "💎", SortOrder: 4},
		},
	}

	for parentSlug, children := range subGenres {
		parent, err := s.genreDAO.FindBySlug(parentSlug)
		if err != nil {
			continue
		}
		for i := range children {
			children[i].ParentID = parent.ID
		}
		if err := s.genreDAO.SeedGenres(children); err != nil {
			log.Printf("[genre] seed sub genres for %s failed: %v", parentSlug, err)
		}
	}

	log.Println("[genre] seed defaults done")
}

// GetGenreTree 获取赛道树
func (s *GenreService) GetGenreTree() ([]model.GenreTree, error) {
	genres, err := s.genreDAO.ListAll()
	if err != nil {
		return nil, err
	}
	return buildTree(genres, 0), nil
}

// buildTree 递归构建赛道树
func buildTree(genres []model.Genre, parentID uint) []model.GenreTree {
	var tree []model.GenreTree
	for _, g := range genres {
		if g.ParentID == parentID {
			node := model.GenreTree{Genre: g}
			node.Children = buildTree(genres, g.ID)
			tree = append(tree, node)
		}
	}
	return tree
}

// GetGenre 获取赛道详情
func (s *GenreService) GetGenre(id uint) (*model.Genre, error) {
	return s.genreDAO.Get(id)
}

// CreateGenre 创建赛道（管理员）
func (s *GenreService) CreateGenre(genre *model.Genre) error {
	return s.genreDAO.Create(genre)
}

// UpdateGenre 更新赛道（管理员）
func (s *GenreService) UpdateGenre(genre *model.Genre) error {
	return s.genreDAO.Update(genre)
}

// DeleteGenre 删除赛道（管理员）
func (s *GenreService) DeleteGenre(id uint) error {
	return s.genreDAO.Delete(id)
}

// SetMemoryGenres 设置记忆的赛道关联
func (s *GenreService) SetMemoryGenres(memoryID uint, genreIDs []uint) error {
	return s.genreDAO.SetMemoryGenres(memoryID, genreIDs)
}

// ListGenresByMemory 获取记忆关联的赛道 ID 列表
func (s *GenreService) ListGenresByMemory(memoryID uint) ([]uint, error) {
	return s.genreDAO.ListByMemory(memoryID)
}

// ListMemoryIDsByGenre 获取某赛道下的所有记忆 ID
func (s *GenreService) ListMemoryIDsByGenre(genreID uint) ([]uint, error) {
	return s.genreDAO.ListMemoryIDsByGenre(genreID)
}
