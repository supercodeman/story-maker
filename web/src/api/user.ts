// web/src/api/user.ts
import request from './request'

export const userAdminApi = {
  // 获取所有用户列表
  listUsers: () => request.get('/admin/users'),
  // 更新用户角色
  updateRole: (uid: number, data: { role: string }) =>
    request.put(`/admin/users/${uid}/role`, data),
  // 管理员设置写手等级
  setWriterLevel: (uid: number, data: { writer_level: string }) =>
    request.put(`/admin/users/${uid}/writer-level`, data),
}
