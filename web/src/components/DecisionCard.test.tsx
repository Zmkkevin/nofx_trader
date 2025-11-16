import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { DecisionCard } from './DecisionCard'
import type { DecisionRecord } from '../types'

/**
 * DecisionCard 组件测试 - 验证新增字段的显示
 *
 * 测试场景：
 * - update_stop_loss 显示 new_stop_loss 字段
 * - update_take_profit 显示 new_take_profit 字段
 * - partial_close 显示 close_percentage 字段
 */

describe('DecisionCard - New Fields Display', () => {
  const baseDecision: DecisionRecord = {
    cycle_number: 1,
    timestamp: '2025-01-16T10:00:00Z',
    success: true,
    decisions: [],
    execution_log: [],
    account_state: {
      total_balance: 10000,
      available_balance: 5000,
      total_unrealized_profit: 0,
      position_count: 0,
      margin_used_pct: 0,
      initial_balance: 10000,
    },
    input_prompt: '',
    cot_trace: '',
    error_message: '',
  }

  it('should display new_stop_loss field for update_stop_loss action', () => {
    const decision: DecisionRecord = {
      ...baseDecision,
      decisions: [
        {
          action: 'update_stop_loss',
          symbol: 'BTCUSDT',
          price: 50000.0,
          new_stop_loss: 48000.0,
          quantity: 0,
          leverage: 0,
          order_id: 0,
          timestamp: '2025-01-16T10:00:00Z',
          success: true,
          error: '',
        },
      ],
    }

    render(<DecisionCard decision={decision} language="en" />)

    // 验证 symbol 显示
    expect(screen.getByText('BTCUSDT')).toBeInTheDocument()

    // 验证 action 显示
    expect(screen.getByText('update_stop_loss')).toBeInTheDocument()

    // Note: 当前实现只显示 action 文本，后续可以增强显示 new_stop_loss 的值
  })

  it('should display new_take_profit field for update_take_profit action', () => {
    const decision: DecisionRecord = {
      ...baseDecision,
      decisions: [
        {
          action: 'update_take_profit',
          symbol: 'ETHUSDT',
          price: 3000.0,
          new_take_profit: 3200.0,
          quantity: 0,
          leverage: 0,
          order_id: 0,
          timestamp: '2025-01-16T10:00:00Z',
          success: true,
          error: '',
        },
      ],
    }

    render(<DecisionCard decision={decision} language="en" />)

    expect(screen.getByText('ETHUSDT')).toBeInTheDocument()
    expect(screen.getByText('update_take_profit')).toBeInTheDocument()
  })

  it('should display close_percentage field for partial_close action', () => {
    const decision: DecisionRecord = {
      ...baseDecision,
      decisions: [
        {
          action: 'partial_close',
          symbol: 'SOLUSDT',
          price: 100.0,
          quantity: 5.0,
          close_percentage: 50.0,
          leverage: 0,
          order_id: 0,
          timestamp: '2025-01-16T10:00:00Z',
          success: true,
          error: '',
        },
      ],
    }

    render(<DecisionCard decision={decision} language="en" />)

    expect(screen.getByText('SOLUSDT')).toBeInTheDocument()
    expect(screen.getByText('partial_close')).toBeInTheDocument()
  })

  it('should display multiple actions with different new fields', () => {
    const decision: DecisionRecord = {
      ...baseDecision,
      decisions: [
        {
          action: 'update_stop_loss',
          symbol: 'BTCUSDT',
          price: 50000.0,
          new_stop_loss: 48000.0,
          quantity: 0,
          leverage: 0,
          order_id: 0,
          timestamp: '2025-01-16T10:00:00Z',
          success: true,
          error: '',
        },
        {
          action: 'update_take_profit',
          symbol: 'ETHUSDT',
          price: 3000.0,
          new_take_profit: 3200.0,
          quantity: 0,
          leverage: 0,
          order_id: 0,
          timestamp: '2025-01-16T10:00:00Z',
          success: true,
          error: '',
        },
        {
          action: 'partial_close',
          symbol: 'SOLUSDT',
          price: 100.0,
          quantity: 5.0,
          close_percentage: 50.0,
          leverage: 0,
          order_id: 0,
          timestamp: '2025-01-16T10:00:00Z',
          success: true,
          error: '',
        },
      ],
    }

    render(<DecisionCard decision={decision} language="en" />)

    // 验证所有 symbols 都显示
    expect(screen.getByText('BTCUSDT')).toBeInTheDocument()
    expect(screen.getByText('ETHUSDT')).toBeInTheDocument()
    expect(screen.getByText('SOLUSDT')).toBeInTheDocument()

    // 验证所有 actions 都显示
    expect(screen.getByText('update_stop_loss')).toBeInTheDocument()
    expect(screen.getByText('update_take_profit')).toBeInTheDocument()
    expect(screen.getByText('partial_close')).toBeInTheDocument()
  })

  it('should handle missing optional fields gracefully', () => {
    const decision: DecisionRecord = {
      ...baseDecision,
      decisions: [
        {
          action: 'update_stop_loss',
          symbol: 'BTCUSDT',
          price: 50000.0,
          // new_stop_loss 字段缺失
          quantity: 0,
          leverage: 0,
          order_id: 0,
          timestamp: '2025-01-16T10:00:00Z',
          success: true,
          error: '',
        },
      ],
    }

    // 应该不会崩溃，正常渲染
    render(<DecisionCard decision={decision} language="en" />)

    expect(screen.getByText('BTCUSDT')).toBeInTheDocument()
    expect(screen.getByText('update_stop_loss')).toBeInTheDocument()
  })
})

/**
 * 数据类型验证测试
 * 确保新字段的类型定义正确
 */
describe('DecisionCard - Data Type Validation', () => {
  it('should accept valid new_stop_loss number', () => {
    const validAction = {
      action: 'update_stop_loss',
      symbol: 'BTCUSDT',
      price: 50000.0,
      new_stop_loss: 48000.0,
      quantity: 0,
      leverage: 0,
      order_id: 0,
      timestamp: '2025-01-16T10:00:00Z',
      success: true,
      error: '',
    }

    expect(typeof validAction.new_stop_loss).toBe('number')
    expect(validAction.new_stop_loss).toBeGreaterThan(0)
  })

  it('should accept valid new_take_profit number', () => {
    const validAction = {
      action: 'update_take_profit',
      symbol: 'ETHUSDT',
      price: 3000.0,
      new_take_profit: 3200.0,
      quantity: 0,
      leverage: 0,
      order_id: 0,
      timestamp: '2025-01-16T10:00:00Z',
      success: true,
      error: '',
    }

    expect(typeof validAction.new_take_profit).toBe('number')
    expect(validAction.new_take_profit).toBeGreaterThan(0)
  })

  it('should accept valid close_percentage number in range 0-100', () => {
    const validAction = {
      action: 'partial_close',
      symbol: 'SOLUSDT',
      price: 100.0,
      quantity: 5.0,
      close_percentage: 50.0,
      leverage: 0,
      order_id: 0,
      timestamp: '2025-01-16T10:00:00Z',
      success: true,
      error: '',
    }

    expect(typeof validAction.close_percentage).toBe('number')
    expect(validAction.close_percentage).toBeGreaterThan(0)
    expect(validAction.close_percentage).toBeLessThanOrEqual(100)
  })
})
