import { useState } from 'react'
import { LANGS, type Lang } from '../i18n'
import { ago, api, navigate, useApp, useData } from '../lib'
import { ErrorLine, Icon, PageHead, SpeakButton, Spinner } from '../ui'

export function Help() {
  const { t, lang, toast } = useApp()
  const faqs = useData<{ id: number; topic: string; q: string; a: string }[]>('/faqs', [lang])
  const tickets = useData<{ id: number; subject: string; status: string; created_at: string }[]>('/tickets')
  const [q, setQ] = useState('')
  const [answer, setAnswer] = useState<{ answer: string; suggest_ticket: boolean } | null>(null)
  const [busy, setBusy] = useState(false)
  const [subject, setSubject] = useState('')
  const [body, setBody] = useState('')
  const [err, setErr] = useState('')

  const ask = async () => {
    setBusy(true); setAnswer(null)
    try { setAnswer(await api('/help/ask', { body: { question: q } })) } catch (e) { toast((e as Error).message) } finally { setBusy(false) }
  }
  const raise = async () => {
    setErr('')
    try { await api('/tickets', { body: { subject, body } }); setSubject(''); setBody(''); toast(t('ticket_sent')); tickets.reload() } catch (e) { setErr((e as Error).message) }
  }

  return (
    <>
      <PageHead kicker={t('help_kicker')} title={t('help_title')}>
        <a className="btn red sm" href="tel:1930">☎ 1930</a>
      </PageHead>
      <div className="grid2" style={{ alignItems: 'start' }}>
        <div className="stack">
          <form className="sheet stack" onSubmit={(e) => { e.preventDefault(); ask() }}>
            <h3>{t('ask_help')}</h3>
            <div className="row">
              <input className="input grow" value={q} onChange={(e) => setQ(e.target.value)} aria-label={t('ask_help')} />
              <button className="btn primary" disabled={busy || !q.trim()}>{busy ? <Spinner /> : t('send')}</button>
            </div>
            {answer && (
              <div className="sheet plain ruled" style={{ lineHeight: '32px' }}>
                <div className="row between"><p>{answer.answer}</p><SpeakButton text={answer.answer} /></div>
                {answer.suggest_ticket && <button type="button" className="btn sm" onClick={() => { setSubject(q); setBody(q) }}>{t('raise_ticket')} →</button>}
              </div>
            )}
          </form>
          <div className="sheet">
            <h3>{t('faq')}</h3>
            {faqs.loading && <Spinner />}
            {faqs.data?.map((f) => (
              <details key={f.id} className="faq">
                <summary>{f.q}</summary>
                <div className="row between" style={{ alignItems: 'flex-start' }}><p>{f.a}</p><SpeakButton text={f.a} /></div>
              </details>
            ))}
          </div>
        </div>
        <div className="stack">
          <form className="sheet stack" onSubmit={(e) => { e.preventDefault(); raise() }}>
            <h3>{t('raise_ticket')}</h3>
            <div className="field"><label htmlFor="subj">{t('subject')}</label><input id="subj" className="input" value={subject} onChange={(e) => setSubject(e.target.value)} maxLength={150} /></div>
            <div className="field"><label htmlFor="tbody">{t('describe')}</label><textarea id="tbody" className="textarea" value={body} onChange={(e) => setBody(e.target.value)} maxLength={3000} /></div>
            <ErrorLine msg={err} />
            <button className="btn primary" style={{ alignSelf: 'flex-start' }} disabled={!subject.trim() || !body.trim()}>{t('submit')}</button>
          </form>
          {tickets.data && tickets.data.length > 0 && (
            <div className="sheet">
              <h3>{t('your_tickets')}</h3>
              <table className="ledger-table"><tbody>
                {tickets.data.map((x) => <tr key={x.id}><td className="mono tiny">#{x.id}</td><td>{x.subject}</td><td><span className={`chip ${x.status === 'open' ? 'yellow' : 'green'}`}>{x.status}</span></td></tr>)}
              </tbody></table>
            </div>
          )}
        </div>
      </div>
    </>
  )
}

export function Settings() {
  const { t, user, setUser, lang, setLang, theme, toggleTheme, toast } = useApp()
  const [name, setName] = useState(user?.name ?? '')
  const [oldPin, setOldPin] = useState('')
  const [newPin, setNewPin] = useState('')
  const [delPin, setDelPin] = useState('')
  const [err, setErr] = useState('')

  const put = async (body: Record<string, unknown>) => {
    setErr('')
    try { const u = await api('/settings', { method: 'PUT', body }); setUser(u); toast(t('updated')); return true } catch (e) { setErr((e as Error).message); return false }
  }

  return (
    <>
      <PageHead kicker={user?.phone ? `+91 ${user.phone}` : ''} title={t('settings_title')} />
      <div className="grid2" style={{ alignItems: 'start' }}>
        <div className="sheet stack">
          <div className="field"><label htmlFor="sname">{t('name')}</label>
            <div className="row"><input id="sname" className="input grow" value={name} onChange={(e) => setName(e.target.value)} /><button className="btn sm" onClick={() => put({ name })}>{t('save')}</button></div>
          </div>
          <div className="field"><label htmlFor="slang">{t('lang')}</label>
            <select id="slang" className="select" value={lang} onChange={(e) => setLang(e.target.value as Lang)}>{LANGS.map((l) => <option key={l.code} value={l.code}>{l.label}</option>)}</select>
          </div>
          <div className="row between"><span style={{ fontWeight: 600 }}>{theme === 'dark' ? t('theme_dark') : t('theme_light')}</span>
            <div className="seg"><button aria-pressed={theme === 'light'} onClick={() => theme !== 'light' && toggleTheme()}><Icon name="sun" size={16} /></button><button aria-pressed={theme === 'dark'} onClick={() => theme !== 'dark' && toggleTheme()}><Icon name="moon" size={16} /></button></div>
          </div>
          <label className="row" style={{ cursor: 'pointer' }}>
            <input type="checkbox" checked={!!user?.prefs?.auto_read} onChange={(e) => put({ prefs: { auto_read: e.target.checked } })} style={{ width: 20, height: 20 }} />
            {t('auto_read')}
          </label>
          <ErrorLine msg={err} />
        </div>
        <div className="stack">
          <form className="sheet stack" onSubmit={async (e) => { e.preventDefault(); if (await put({ old_pin: oldPin, new_pin: newPin })) { setOldPin(''); setNewPin('') } }}>
            <h3>{t('change_pin')}</h3>
            <div className="row">
              <input className="input pin" type="password" inputMode="numeric" maxLength={4} placeholder={t('old_pin')} aria-label={t('old_pin')} value={oldPin} onChange={(e) => setOldPin(e.target.value.replace(/\D/g, ''))} />
              <input className="input pin" type="password" inputMode="numeric" maxLength={4} placeholder={t('new_pin')} aria-label={t('new_pin')} value={newPin} onChange={(e) => setNewPin(e.target.value.replace(/\D/g, ''))} />
            </div>
            <button className="btn sm" style={{ alignSelf: 'flex-start' }} disabled={oldPin.length !== 4 || newPin.length !== 4}>{t('save')}</button>
          </form>
          <div className="sheet stack" style={{ borderColor: 'var(--stamp)' }}>
            <h3 style={{ color: 'var(--stamp)' }}>{t('danger')}</h3>
            <button className="btn sm" style={{ alignSelf: 'flex-start' }} onClick={async () => { await api('/memories/all', { method: 'DELETE' }); toast(t('updated')) }}>{t('clear_memory')}</button>
            <p className="small muted">{t('delete_confirm')}</p>
            <div className="row">
              <input className="input pin" type="password" inputMode="numeric" maxLength={4} aria-label={t('pin')} value={delPin} onChange={(e) => setDelPin(e.target.value.replace(/\D/g, ''))} />
              <button className="btn red sm" disabled={delPin.length !== 4} onClick={async () => {
                try { await api('/account', { method: 'DELETE', body: { pin: delPin } }); setUser(null); navigate('/') } catch (e) { setErr((e as Error).message) }
              }}>{t('delete_account')}</button>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}

type Overview = {
  counts: Record<string, number>
  sources: { source: string; ok: boolean; count: number; error: string; ran_at: string }[]
  ai: { kind: string; calls: number; avg_ms: number; p90_ms: number; tokens: number; failed: number }[]
  runs: { source: string; ok: boolean; count: number; error: string; ms: number; ran_at: string }[]
}

export function Admin() {
  const { t, lang, toast } = useApp()
  const d = useData<Overview>('/admin/overview')
  const tickets = useData<{ id: number; subject: string; body: string; status: string; created_at: string; name: string }[]>('/tickets?all=1')
  const [busy, setBusy] = useState(false)
  const refresh = async () => {
    setBusy(true)
    try { await api('/grants/refresh', { method: 'POST' }); d.reload() } catch (e) { toast((e as Error).message) } finally { setBusy(false) }
  }
  return (
    <>
      <PageHead kicker={t('admin_kicker')} title={t('admin_title')}>
        <button className="btn primary sm" onClick={refresh} disabled={busy}>{busy ? <Spinner /> : <Icon name="refresh" size={16} />}{t('refresh')}</button>
      </PageHead>
      {d.loading && !d.data && <Spinner />}
      {d.data && (
        <div className="stack lg">
          <div className="grid3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))' }}>
            {Object.entries(d.data.counts).sort().map(([k, v]) => <div key={k} className="stat"><div className="label">{k.replace(/_/g, ' ')}</div><div className="value">{v}</div></div>)}
          </div>
          <div className="grid2" style={{ alignItems: 'start' }}>
            <div className="sheet">
              <h3>{t('sources_health')}</h3>
              <table className="ledger-table"><tbody>
                {d.data.sources.map((s) => (
                  <tr key={s.source}><td className="small">{s.source.replace(/^https?:\/\//, '')}<div className="tiny faint">{s.error}</div></td><td><span className={`chip ${s.ok ? 'green' : 'red'}`}>{s.ok ? 'OK' : 'FAIL'}</span></td><td className="amt mono">{s.count}</td><td className="tiny faint">{ago(s.ran_at, lang)}</td></tr>
                ))}
              </tbody></table>
            </div>
            <div className="sheet">
              <h3>AI calls · 7 days</h3>
              <table className="ledger-table">
                <thead><tr><th>kind</th><th className="amt">calls</th><th className="amt">avg</th><th className="amt">p90</th><th className="amt">tokens</th></tr></thead>
                <tbody>
                  {d.data.ai.map((a) => <tr key={a.kind}><td>{a.kind}{a.failed > 0 && <span className="chip red" style={{ marginLeft: 4 }}>{a.failed} ✗</span>}</td><td className="amt mono">{a.calls}</td><td className="amt mono">{(a.avg_ms / 1000).toFixed(1)}s</td><td className="amt mono">{(a.p90_ms / 1000).toFixed(1)}s</td><td className="amt mono">{a.tokens.toLocaleString('en-IN')}</td></tr>)}
                </tbody>
              </table>
            </div>
          </div>
          <div className="sheet">
            <h3>Scraper runs</h3>
            <div style={{ overflowX: 'auto' }}>
              <table className="ledger-table"><tbody>
                {d.data.runs.map((r, i) => <tr key={i}><td className="tiny faint mono" style={{ whiteSpace: 'nowrap' }}>{ago(r.ran_at, lang)}</td><td className="small">{r.source.replace(/^https?:\/\//, '')}</td><td><span className={`chip ${r.ok ? 'green' : 'red'}`}>{r.ok ? r.count : 'FAIL'}</span></td><td className="amt mono tiny">{r.ms}ms</td><td className="tiny faint">{r.error}</td></tr>)}
              </tbody></table>
            </div>
          </div>
          {tickets.data && tickets.data.length > 0 && (
            <div className="sheet">
              <h3>Tickets</h3>
              <table className="ledger-table"><tbody>
                {tickets.data.map((x) => <tr key={x.id}><td className="mono tiny">#{x.id}</td><td><strong>{x.subject}</strong><div className="small muted">{x.body}</div></td><td className="small">{x.name}</td><td className="tiny faint">{x.created_at}</td></tr>)}
              </tbody></table>
            </div>
          )}
        </div>
      )}
    </>
  )
}
