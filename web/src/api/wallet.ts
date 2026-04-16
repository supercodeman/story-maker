// web/src/api/wallet.ts
import request from './request'

// ========== 类型定义 ==========

export interface UserWallet {
  id: number
  user_id: number
  balance: number
  frozen_balance: number
  total_income: number
  total_spent: number
  created_at: string
  updated_at: string
}

export interface WalletTransaction {
  id: number
  user_id: number
  type: string
  amount: number
  balance: number
  ref_type: string
  ref_id: number
  description: string
  created_at: string
}

// 流水类型标签
export const txTypeLabels: Record<string, string> = {
  recharge: '充值',
  purchase: '购买',
  income: '收入',
  withdraw: '提现',
  refund: '退款',
}

// ========== API ==========

export const walletApi = {
  // 钱包信息
  getWallet: () =>
    request.get('/wallet'),

  // 流水列表
  listTransactions: (params?: { page?: number; page_size?: number }) =>
    request.get('/wallet/transactions', { params }),

  // 充值
  recharge: (data: { user_id: number; amount: number }) =>
    request.post('/wallet/recharge', data),
}
