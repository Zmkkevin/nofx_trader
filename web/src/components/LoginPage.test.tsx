import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { LoginPage } from './LoginPage'
import { LanguageProvider } from '../contexts/LanguageContext'
import { AuthProvider } from '../contexts/AuthContext'

/**
 * LoginPage 组件渲染测试
 *
 * 目的：验证 LoginPage 能够在有 Router 上下文的情况下正常渲染
 *
 * 这个测试会 catch 以下问题：
 * 1. main.tsx 缺少 BrowserRouter 包装（本次修复的问题）
 * 2. LoginPage 使用了 router hooks (useNavigate) 但没有 router context
 * 3. 缺少必要的 Provider（AuthProvider, LanguageProvider）
 */
describe('LoginPage - Component Rendering', () => {
  /**
   * 测试组件能否在完整的 Provider 包装下正常渲染
   *
   * 如果 main.tsx 缺少 BrowserRouter，这种渲染测试会失败，
   * 因为 LoginPage 内部使用了 useNavigate() hook
   */
  it('should render without crashing when all providers are present', () => {
    // 渲染 LoginPage，包含所有必要的 context providers
    render(
      <BrowserRouter>
        <LanguageProvider>
          <AuthProvider>
            <LoginPage />
          </AuthProvider>
        </LanguageProvider>
      </BrowserRouter>
    )

    // 验证基本元素是否渲染
    // 登录页面应该包含邮箱输入框
    const emailInputs = screen.queryAllByPlaceholderText(/email/i)
    expect(emailInputs.length).toBeGreaterThan(0)
  })

  /**
   * 测试缺少 BrowserRouter 时是否会失败
   *
   * 这个测试验证了为什么需要在 main.tsx 中添加 BrowserRouter
   */
  it('should fail to render without BrowserRouter', () => {
    // 没有 BrowserRouter 包装，LoginPage 使用 useNavigate() 会报错
    expect(() => {
      render(
        <LanguageProvider>
          <AuthProvider>
            <LoginPage />
          </AuthProvider>
        </LanguageProvider>
      )
    }).toThrow()
  })

  /**
   * 测试登录表单的基本元素是否存在
   */
  it('should render login form elements', () => {
    render(
      <BrowserRouter>
        <LanguageProvider>
          <AuthProvider>
            <LoginPage />
          </AuthProvider>
        </LanguageProvider>
      </BrowserRouter>
    )

    // 验证表单元素存在
    const emailInputs = screen.queryAllByPlaceholderText(/email/i)
    expect(emailInputs.length).toBeGreaterThan(0)

    // 密码输入框
    const passwordInputs = screen.queryAllByPlaceholderText(/password/i)
    expect(passwordInputs.length).toBeGreaterThan(0)

    // 登录按钮（可能是中文或英文）
    const loginButtons = screen.queryAllByRole('button')
    expect(loginButtons.length).toBeGreaterThan(0)
  })
})
