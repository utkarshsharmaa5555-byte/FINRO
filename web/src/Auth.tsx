import { useState } from 'react'
import { LANGS, type Lang } from './i18n'
import { api, Link, navigate, useApp, type User } from './lib'
import { demoAccounts } from './Landing'
import { ErrorLine, LangSelect, Spinner, ThemeToggle } from './ui'

export default function Auth({ mode }: { mode: 'login' | 'signup' }) {
  const { t, setUser, lang } = useApp()
  const [phone, setPhone] = useState('')
  const [pin, setPin] = useState('')
  const [name, setName] = useState('')
  const [business, setBusiness] = useState('')
  const [pickLang, setPickLang] = useState<Lang>(lang)
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async (p = phone, n = pin) => {
    setErr('')
    setBusy(true)
    try {
      const u = mode === 'login'
        ? await api<User>('/auth/login', { body: { phone: p, pin: n } })
        : await api<User>('/auth/signup', { body: { phone: p, pin: n, name, business, lang: pickLang } })
      setUser(u)
      navigate('/app')
    } catch (e) {
      setErr((e as Error).message)
    } finally { setBusy(false) }
  }

  return (
    <div className="auth">
      <div className="stack" style={{ width: '100%', maxWidth: 440 }}>
        <div className="row between">
          <Link to="/" className="brand" style={{ padding: 0 }}>
            <span className="brand-mark">F</span><span className="brand-name" style={{ color: 'var(--ink)' }}>Finro</span>
          </Link>
          <div className="row" style={{ gap: 4 }}><LangSelect compact /><ThemeToggle /></div>
        </div>
        <form className="sheet stack" onSubmit={(e) => { e.preventDefault(); submit() }}>
          <h1 className="display" style={{ fontSize: '2rem' }}>{mode === 'login' ? t('login') : t('signup')}</h1>
          {mode === 'signup' && (
            <>
              <div className="field"><label htmlFor="name">{t('name')}</label><input id="name" className="input" value={name} onChange={(e) => setName(e.target.value)} autoComplete="name" required maxLength={80} /></div>
              <div className="field"><label htmlFor="biz">{t('business')}</label><input id="biz" className="input" value={business} onChange={(e) => setBusiness(e.target.value)} maxLength={80} /></div>
              <div className="field">
                <label htmlFor="plang">{t('lang')}</label>
                <select id="plang" className="select" value={pickLang} onChange={(e) => setPickLang(e.target.value as Lang)}>
                  {LANGS.map((l) => <option key={l.code} value={l.code}>{l.label}</option>)}
                </select>
              </div>
            </>
          )}
          <div className="field">
            <label htmlFor="phone">{t('phone')}</label>
            <div className="row" style={{ gap: 6 }}>
              <span className="mono muted">+91</span>
              <input id="phone" className="input mono" inputMode="numeric" pattern="[0-9]{10}" maxLength={10} value={phone}
                onChange={(e) => setPhone(e.target.value.replace(/\D/g, ''))} autoComplete="tel-national" required />
            </div>
          </div>
          <div className="field">
            <label htmlFor="pin">{t('pin')}</label>
            <input id="pin" className="input pin" type="password" inputMode="numeric" pattern="[0-9]{4}" maxLength={4} value={pin}
              onChange={(e) => setPin(e.target.value.replace(/\D/g, ''))} autoComplete={mode === 'login' ? 'current-password' : 'new-password'} required />
          </div>
          <ErrorLine msg={err} />
          <button className="btn primary" disabled={busy}>{busy ? <Spinner /> : mode === 'login' ? t('login') : t('signup')}</button>
          <p className="small muted" style={{ textAlign: 'center' }}>
            {mode === 'login' ? <>{t('new_here')} <Link to="/signup">{t('signup')}</Link></> : <>{t('have_account')} <Link to="/login">{t('login')}</Link></>}
          </p>
        </form>
        {mode === 'login' && (
          <div className="sheet plain stack" style={{ gap: 8 }}>
            <div className="caps muted">{t('demo_accounts')}</div>
            {demoAccounts.map((d) => (
              <button key={d.phone} className="persona" disabled={busy} onClick={() => { setPhone(d.phone); setPin(d.pin); submit(d.phone, d.pin) }}>
                <span className="avatar">{d.name[0]}</span>
                <span className="grow"><strong>{d.name}</strong><div className="tiny muted">{d.role}</div></span>
                <span className="mono tiny muted">{d.phone} · {d.pin}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
