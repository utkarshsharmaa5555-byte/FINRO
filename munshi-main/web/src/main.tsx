import { StrictMode, useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import './theme.css'
import { api, AppProvider, Link, navigate, useApp, useRoute, type User } from './lib'
import { Icon, LangSelect, Spinner, ThemeToggle } from './ui'
import Landing from './Landing'
import Auth from './Auth'
import Assistant from './Assistant'
import { Fraud, QRPage, Standee } from './pages/Safety'
import { Khata, Profile } from './pages/Business'
import { Apply, DraftEditor, Funding, Matching } from './pages/Funding'
import { Admin, Help, Settings } from './pages/Desk'
import { OverviewPage } from './pages/Stock'
import { BlueprintPage } from './pages/Blueprint'
import type { Key } from './i18n'

type NavItem = { to: string; label: Key; icon: string; admin?: boolean }
const groups: { label: Key; items: NavItem[] }[] = [
  { label: 'grp_daily', items: [
    { to: '/app', label: 'nav_assistant', icon: 'chat' },
    { to: '/app/overview', label: 'nav_overview', icon: 'dash' },
    { to: '/app/khata', label: 'nav_khata', icon: 'book' },
    { to: '/app/fraud', label: 'nav_fraud', icon: 'shield' },
    { to: '/app/qr', label: 'nav_qr', icon: 'qr' },
  ] },
  { label: 'grp_grow', items: [
    { to: '/app/blueprint', label: 'nav_blueprint', icon: 'flow' },
    { to: '/app/funding', label: 'nav_funding', icon: 'coins' },
    { to: '/app/matching', label: 'nav_matching', icon: 'target' },
    { to: '/app/apply', label: 'nav_apply', icon: 'file' },
  ] },
  { label: 'grp_you', items: [
    { to: '/app/profile', label: 'nav_profile', icon: 'user' },
    { to: '/app/help', label: 'nav_help', icon: 'help' },
    { to: '/app/settings', label: 'nav_settings', icon: 'gear' },
    { to: '/app/admin', label: 'nav_admin', icon: 'radar', admin: true },
  ] },
]

function isActive(path: string, to: string) {
  return to === '/app' ? path === '/app' : path.startsWith(to)
}

function NavLinks({ path, onPick }: { path: string; onPick?: () => void }) {
  const { t, user } = useApp()
  return (
    <nav className="nav" aria-label="Main">
      {groups.map((g) => (
        <div key={g.label}>
          <div className="nav-group">{t(g.label)}</div>
          {g.items.filter((i) => !i.admin || user?.role === 'admin').map((i) => (
            <Link key={i.to} to={i.to} aria-current={isActive(path, i.to) ? 'page' : undefined} onClick={onPick}>
              <Icon name={i.icon} />{t(i.label)}
            </Link>
          ))}
        </div>
      ))}
    </nav>
  )
}

function Shell({ path }: { path: string }) {
  const { t, user, setUser } = useApp()
  const [drawer, setDrawer] = useState(false)
  const logout = async () => { await api('/auth/logout', { method: 'POST' }).catch(() => {}); setUser(null); navigate('/') }

  let page
  const m = (re: RegExp) => path.match(re)
  let mm: RegExpMatchArray | null
  if (path === '/app') page = <Assistant />
  else if ((mm = m(/^\/app\/c\/(\d+)$/))) page = <Assistant sessionId={Number(mm[1])} />
  else if (path === '/app/overview') page = <OverviewPage />
  else if (path === '/app/blueprint') page = <BlueprintPage />
  else if (path === '/app/fraud') page = <Fraud />
  else if ((mm = m(/^\/app\/qr\/(\d+)\/standee$/))) page = <Standee id={Number(mm[1])} />
  else if (path === '/app/qr') page = <QRPage />
  else if (path === '/app/khata') page = <Khata />
  else if (path === '/app/profile') page = <Profile />
  else if (path === '/app/funding') page = <Funding />
  else if (path === '/app/matching') page = <Matching />
  else if ((mm = m(/^\/app\/apply\/(\d+)$/))) page = <DraftEditor id={Number(mm[1])} />
  else if (path === '/app/apply') page = <Apply />
  else if (path === '/app/help') page = <Help />
  else if (path === '/app/settings') page = <Settings />
  else if (path === '/app/admin' && user?.role === 'admin') page = <Admin />
  else page = <Assistant />

  const bottom: NavItem[] = [
    { to: '/app', label: 'nav_assistant', icon: 'chat' },
    { to: '/app/overview', label: 'nav_overview', icon: 'dash' },
    { to: '/app/khata', label: 'nav_khata', icon: 'book' },
    { to: '/app/funding', label: 'nav_funding', icon: 'coins' },
  ]

  return (
    <div className="shell">
      <aside className="spine">
        <Link to="/app" className="brand">
          <span className="brand-mark">F</span>
          <span><span className="brand-name">Finro</span><br /><span className="brand-sub">{t('tagline')}</span></span>
        </Link>
        <NavLinks path={path} />
        <div className="spine-foot">
          <div style={{ fontWeight: 600, color: '#fbeccb' }}>{user?.name}</div>
          <div className="mono tiny">+91 {user?.phone}</div>
          <button className="btn ghost sm" style={{ color: '#f6e7c8', paddingLeft: 0 }} onClick={logout}>{t('logout')}</button>
        </div>
      </aside>
      <div className="main">
        <div className="topbar no-print">
          <Link to="/app" className="brand" style={{ padding: 0 }} aria-label="Finro home">
            <span className="brand-mark" style={{ width: 32, height: 32, fontSize: '1.05rem' }}>F</span>
          </Link>
          <div className="grow" />
          <LangSelect compact />
          <ThemeToggle />
        </div>
        <main className="page" id="main">{page}</main>
      </div>
      <nav className="bottomnav no-print" aria-label="Quick">
        {bottom.map((i) => (
          <Link key={i.to} to={i.to} aria-current={isActive(path, i.to) ? 'page' : undefined}><Icon name={i.icon} />{t(i.label)}</Link>
        ))}
        <button onClick={() => setDrawer(true)} aria-haspopup="dialog"><Icon name="menu" />{t('nav_more')}</button>
      </nav>
      {drawer && (
        <>
          <div className="drawer-backdrop" onClick={() => setDrawer(false)} />
          <div className="drawer" role="dialog" aria-modal="true" aria-label={t('nav_more')}>
            <div className="row between" style={{ color: '#f6e7c8', marginBottom: 8 }}>
              <strong>{user?.name}</strong>
              <button className="btn ghost sm icon" style={{ color: '#f6e7c8' }} onClick={() => setDrawer(false)} aria-label={t('cancel')}><Icon name="x" /></button>
            </div>
            <NavLinks path={path} onPick={() => setDrawer(false)} />
            <button className="btn sm" style={{ marginTop: 12 }} onClick={logout}>{t('logout')}</button>
          </div>
        </>
      )}
    </div>
  )
}

function Root() {
  const path = useRoute()
  const { user, setUser } = useApp()
  const [checked, setChecked] = useState(false)

  useEffect(() => {
    api<User>('/me').then(setUser).catch(() => {}).finally(() => setChecked(true))
  }, [setUser])

  useEffect(() => {
    if (!checked) return
    if (path.startsWith('/app') && !user) navigate('/login')
    if ((path === '/login' || path === '/signup') && user) navigate('/app')
  }, [checked, path, user])

  if (path === '/' || path === '') return <Landing />
  if (!checked) return <div className="auth"><Spinner /></div>
  if (path === '/login' || path === '/signup') return <Auth mode={path === '/signup' ? 'signup' : 'login'} />
  if (path.startsWith('/app') && user) return <Shell path={path} />
  return <Landing />
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AppProvider>
      <Root />
    </AppProvider>
  </StrictMode>,
)
