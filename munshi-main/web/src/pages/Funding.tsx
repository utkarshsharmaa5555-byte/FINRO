import { useEffect, useState } from 'react'
import { ago, api, Link, Md, navigate, useApp, useData } from '../lib'
import { BlueprintView, Empty, ErrorLine, FitCard, GrantCard, Icon, PageHead, SpeakButton, Spinner, type Blueprint, type Fit, type Grant } from '../ui'

// ---------- Scheme finder (used in chat and on Live funding) ----------

const FIELD_LABEL: Record<string, Record<string, string>> = {
  en: { state: 'Which state is your business in?', gender: 'Gender', age: 'Age', social_category: 'Social category', business_domain: 'Business domain you want to explore', business_stage: 'Stage of business', needs: 'What help do you need? (tap all)' },
  hi: { state: 'आपका व्यापार किस राज्य में है?', gender: 'लिंग', age: 'उम्र', social_category: 'सामाजिक वर्ग', business_domain: 'किस तरह का व्यापार', business_stage: 'व्यापार की स्थिति', needs: 'किस मदद की ज़रूरत है? (सभी चुनें)' },
  te: { state: 'మీ వ్యాపారం ఏ రాష్ట్రంలో ఉంది?', gender: 'లింగం', age: 'వయసు', social_category: 'సామాజిక వర్గం', business_domain: 'ఏ రకమైన వ్యాపారం', business_stage: 'వ్యాపార దశ', needs: 'ఏ సహాయం కావాలి? (అన్నీ ఎంచుకోండి)' },
  ta: { state: 'உங்கள் வணிகம் எந்த மாநிலத்தில்?', gender: 'பாலினம்', age: 'வயது', social_category: 'சமூகப் பிரிவு', business_domain: 'எந்த வகை வணிகம்', business_stage: 'வணிக நிலை', needs: 'என்ன உதவி வேண்டும்? (அனைத்தையும் தேர்வு)' },
}
const FINDER: Record<string, { title: string; go: string; read: string; get: string; how: string; confirm: string; suggestions: string; docs: string; linkok: string; considered: string; details: string }> = {
  en: { title: 'Find schemes for me', go: 'Show my schemes', read: 'Read full rules on official site', get: 'What you get', how: 'How to apply', confirm: 'Confirm on the site', suggestions: 'Finro suggests', docs: 'Keep these ready', linkok: 'link checked', considered: 'schemes checked for you', details: 'Your details' },
  hi: { title: 'मेरे लिए योजनाएँ खोजें', go: 'मेरी योजनाएँ दिखाएँ', read: 'सरकारी साइट पर पूरे नियम पढ़ें', get: 'आपको क्या मिलेगा', how: 'आवेदन कैसे करें', confirm: 'साइट पर पुष्टि करें', suggestions: 'फिनरो की सलाह', docs: 'ये तैयार रखें', linkok: 'लिंक जाँचा गया', considered: 'योजनाएँ आपके लिए जाँची गईं', details: 'आपकी जानकारी' },
  te: { title: 'నాకు పథకాలు వెతకండి', go: 'నా పథకాలు చూపించు', read: 'అధికారిక సైట్‌లో పూర్తి నియమాలు చదవండి', get: 'మీకు ఏం లభిస్తుంది', how: 'ఎలా దరఖాస్తు చేయాలి', confirm: 'సైట్‌లో నిర్ధారించుకోండి', suggestions: 'ఫిన్‌రో సూచనలు', docs: 'ఇవి సిద్ధంగా ఉంచండి', linkok: 'లింక్ చెక్ చేశాం', considered: 'పథకాలు మీ కోసం చెక్ చేశాం', details: 'మీ వివరాలు' },
  ta: { title: 'எனக்கான திட்டங்களைக் கண்டுபிடி', go: 'என் திட்டங்களைக் காட்டு', read: 'அதிகாரப்பூர்வ தளத்தில் முழு விதிகளைப் படியுங்கள்', get: 'உங்களுக்கு என்ன கிடைக்கும்', how: 'எப்படி விண்ணப்பிப்பது', confirm: 'தளத்தில் உறுதிப்படுத்துங்கள்', suggestions: 'ஃபின்ரோ பரிந்துரைகள்', docs: 'இவற்றைத் தயாராக வையுங்கள்', linkok: 'இணைப்பு சரிபார்க்கப்பட்டது', considered: 'திட்டங்கள் உங்களுக்காகச் சரிபார்க்கப்பட்டன', details: 'உங்கள் விவரங்கள்' },
}

type Intake = { missing: string[]; profile: Record<string, any>; options: Record<string, string[]> }

export function SchemeIntakeForm({ intake, onDone, submitLabel }: { intake: Intake; onDone: (profile: Record<string, any>) => void; submitLabel?: string }) {
  const { lang, t } = useApp()
  const fields = ['state', 'gender', 'age', 'social_category', 'business_domain', 'business_stage', 'needs']
  const [vals, setVals] = useState<Record<string, any>>(() => Object.fromEntries(fields.map((f) => [f, intake.profile[f] ?? (f === 'needs' ? [] : '')])))
  const [busy, setBusy] = useState(false)
  const [done, setDone] = useState(false)
  const [err, setErr] = useState('')
  const show = intake.missing.length ? fields.filter((f) => intake.missing.includes(f)) : fields
  const known = fields.filter((f) => !show.includes(f) && intake.profile[f])
  const complete = fields.every((f) => (f === 'needs' ? vals.needs.length > 0 : String(vals[f] ?? '').trim() !== ''))

  const submit = async () => {
    setBusy(true); setErr('')
    try {
      const body = { ...vals, age: Number(vals.age) }
      await api('/profile', { method: 'PUT', body })
      setDone(true)
      onDone(body)
    } catch (e) { setErr((e as Error).message) } finally { setBusy(false) }
  }

  if (done) return <div className="sheet row"><span className="stamp green thump">✓ {t('saved')}</span></div>
  return (
    <form className="sheet stack" onSubmit={(e) => { e.preventDefault(); submit() }}>
      <div className="caps muted">{FINDER[lang].details}</div>
      {known.length > 0 && <div className="row wrap" style={{ gap: 4 }}>{known.map((f) => <span key={f} className="chip green">✓ {Array.isArray(intake.profile[f]) ? intake.profile[f].join(', ') : String(intake.profile[f])}</span>)}</div>}
      {show.map((f) => (
        <fieldset key={f} style={{ border: 0, padding: 0, margin: 0 }}>
          <legend style={{ fontWeight: 600, marginBottom: 6 }}>{FIELD_LABEL[lang][f]}{intake.missing.includes(f) && <span style={{ color: 'var(--stamp)' }}> *</span>}</legend>
          {f === 'age' ? (
            <input className="input mono" style={{ maxWidth: 120 }} inputMode="numeric" value={vals.age} onChange={(e) => setVals({ ...vals, age: e.target.value.replace(/\D/g, '').slice(0, 2) })} aria-label={FIELD_LABEL[lang].age} />
          ) : f === 'state' ? (
            <select className="select" value={vals.state} onChange={(e) => setVals({ ...vals, state: e.target.value })} aria-label={FIELD_LABEL[lang].state}>
              <option value="">—</option>{intake.options.state.map((o) => <option key={o}>{o}</option>)}
            </select>
          ) : (
            <div className="row wrap" style={{ gap: 6 }}>
              {intake.options[f].map((o) => {
                const on = f === 'needs' ? vals.needs.includes(o) : vals[f] === o
                return (
                  <button type="button" key={o} className="suggestion" aria-pressed={on}
                    style={on ? { background: 'var(--ink)', color: 'var(--paper)', borderStyle: 'solid', borderColor: 'var(--ink)' } : undefined}
                    onClick={() => setVals({ ...vals, [f]: f === 'needs' ? (on ? vals.needs.filter((x: string) => x !== o) : [...vals.needs, o]) : o })}>
                    {o}
                  </button>
                )
              })}
            </div>
          )}
        </fieldset>
      ))}
      <ErrorLine msg={err} />
      <button className="btn primary" disabled={busy || !complete} style={{ alignSelf: 'flex-start' }}>{busy ? <Spinner /> : <Icon name="target" size={18} />}{submitLabel ?? FINDER[lang].go}</button>
    </form>
  )
}

type Rec = { grant_id: number; fit: 'high' | 'medium'; why_for_you: string; what_you_get: string; how_to_apply: string[]; check_on_site: string[]; grant: Grant }
export type SchemeResult = { recommendations: Rec[]; suggestions: string[]; documents_to_keep_ready: string[]; considered: number }

export function SchemeRecs({ res }: { res: SchemeResult }) {
  const { lang, t } = useApp()
  const c = FINDER[lang]
  const [drafting, setDrafting] = useState(0)
  const draft = async (id: number) => {
    setDrafting(id)
    try { const d = await api('/drafts', { body: { grant_id: id } }); navigate(`/app/apply/${d.id}`) } catch { setDrafting(0) }
  }
  return (
    <div className="stack">
      <p className="tiny faint mono">✓ {res.considered} {c.considered}</p>
      {res.recommendations.map((r) => (
        <article key={r.grant_id} className="sheet stack" style={{ gap: '0.55rem' }}>
          <div className="row between wrap" style={{ alignItems: 'flex-start' }}>
            <span className="caps faint">{r.grant.source}</span>
            <span className={`stamp ${r.fit === 'high' ? 'green' : 'yellow'}`}>{r.fit === 'high' ? t('strong') : t('possible')}</span>
          </div>
          <h3 style={{ fontSize: '1.1rem' }}>{r.grant.title}</h3>
          <p className="small muted">{r.grant.summary}</p>
          <p><strong style={{ color: 'var(--ink)' }}>→</strong> {r.why_for_you}</p>
          <div className="grid2" style={{ gap: '0.6rem' }}>
            <div><div className="caps muted">{c.get}</div><p className="small">{r.what_you_get}</p></div>
            <div><div className="caps muted">{c.how}</div><ol className="do-list small">{r.how_to_apply.map((s, i) => <li key={i}>{s}</li>)}</ol></div>
          </div>
          {r.check_on_site.length > 0 && (
            <div><div className="caps muted">{c.confirm}</div><ul className="flag-list small" style={{ margin: 0 }}>{r.check_on_site.map((s, i) => <li key={i}>{s}</li>)}</ul></div>
          )}
          <div className="row between wrap">
            <div className="row wrap" style={{ gap: 6 }}>
              <a className="btn primary sm" href={r.grant.source_url.split('#')[0]} target="_blank" rel="noopener noreferrer"><Icon name="ext" size={15} />{c.read}</a>
              <button className="btn sm" disabled={drafting === r.grant_id} onClick={() => draft(r.grant_id)}>{drafting === r.grant_id ? <Spinner /> : <Icon name="file" size={15} />}{t('draft_app')}</button>
            </div>
            <span className="tiny faint">{r.grant.link_ok ? `✓ ${c.linkok} · ` : ''}{ago(r.grant.fetched_at, lang)}</span>
          </div>
          <div className="row" style={{ justifyContent: 'flex-end' }}><SpeakButton text={[r.grant.title, r.why_for_you, r.what_you_get, ...r.how_to_apply].join('. ')} /></div>
        </article>
      ))}
      {(res.suggestions.length > 0 || res.documents_to_keep_ready.length > 0) && (
        <div className="sheet ruled grid2" style={{ lineHeight: '32px' }}>
          <div><div className="caps muted">{c.suggestions}</div><ol className="do-list">{res.suggestions.map((s, i) => <li key={i}>{s}</li>)}</ol></div>
          <div><div className="caps muted">{c.docs}</div><ul style={{ margin: 0, paddingLeft: '1.1rem' }}>{res.documents_to_keep_ready.map((s, i) => <li key={i}>☐ {s}</li>)}</ul></div>
        </div>
      )}
    </div>
  )
}

function SchemeFinder() {
  const { lang } = useApp()
  const c = FINDER[lang]
  const [intake, setIntake] = useState<Intake | null>(null)
  const [res, setRes] = useState<SchemeResult | null>(null)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  const [open, setOpen] = useState(false)

  const start = async () => {
    setOpen(true); setErr('')
    const i = await api<Intake>('/schemes/intake')
    setIntake(i)
    if (i.missing.length === 0) find()
  }
  const find = async () => {
    setBusy(true); setErr(''); setRes(null)
    try { setRes(await api<SchemeResult>('/schemes/find', { body: {} })) } catch (e) { setErr((e as Error).message) } finally { setBusy(false) }
  }
  return (
    <section className="stack" style={{ marginBottom: '1.5rem' }}>
      {!open && <button className="btn red" style={{ alignSelf: 'flex-start' }} onClick={start}><Icon name="target" size={18} />{c.title}</button>}
      {open && intake && !busy && !res && <SchemeIntakeForm intake={{ ...intake, missing: intake.missing.length ? intake.missing : [] }} onDone={find} />}
      {busy && <div className="sheet"><Spinner /></div>}
      <ErrorLine msg={err} />
      {res && <SchemeRecs res={res} />}
    </section>
  )
}

// ---------- Live funding ----------

type Health = { source: string; ok: boolean; count: number; error: string; ran_at: string; last_ok: string | null }

export function Funding() {
  const { t, lang, user, toast } = useApp()
  const d = useData<{ grants: Grant[]; sources: Health[]; refreshed_now: boolean }>('/grants', [lang])
  const [filter, setFilter] = useState('all')
  const [refreshing, setRefreshing] = useState(false)
  const sources = [...new Set(d.data?.grants.map((g) => g.source) ?? [])]
  const shown = d.data?.grants.filter((g) => filter === 'all' || g.source === filter) ?? []

  const refresh = async () => {
    setRefreshing(true)
    try { await api('/grants/refresh', { method: 'POST' }); d.reload() } catch (e) { toast((e as Error).message) } finally { setRefreshing(false) }
  }

  return (
    <>
      <PageHead kicker={t('funding_kicker')} title={t('funding_title')}>
        {user?.role === 'admin' && <button className="btn sm" onClick={refresh} disabled={refreshing}>{refreshing ? <Spinner /> : <Icon name="refresh" size={16} />}{t('refresh')}</button>}
      </PageHead>
      <SchemeFinder />
      {d.loading && !d.data && <div className="grid2">{[1, 2, 3, 4].map((i) => <div key={i} className="skeleton" style={{ height: 220 }} />)}</div>}
      <ErrorLine msg={d.error} />
      {d.data && (
        <>
          <div className="sheet plain" style={{ marginBottom: '1rem' }}>
            <div className="caps muted" style={{ marginBottom: 6 }}>{t('sources_health')}</div>
            <div className="row wrap" style={{ gap: 6 }}>
              {d.data.sources.map((s) => (
                <span key={s.source} className={`chip ${s.ok ? 'green' : 'red'}`} title={s.error || undefined}>
                  {s.ok ? '●' : '○'} {s.source.replace(/^https?:\/\/(www\.)?/, '').replace(/\/.*$/, '')} · {s.count} · {ago(s.ran_at, lang)}
                </span>
              ))}
            </div>
          </div>
          <div className="seg" role="group" style={{ marginBottom: '1rem', flexWrap: 'wrap' }}>
            <button aria-pressed={filter === 'all'} onClick={() => setFilter('all')}>{t('all')} ({d.data.grants.length})</button>
            {sources.map((s) => <button key={s} aria-pressed={filter === s} onClick={() => setFilter(s)}>{s}</button>)}
          </div>
          <div className="grid2">{shown.map((g) => <GrantCard key={g.id} g={g} />)}</div>
        </>
      )}
    </>
  )
}

// ---------- Matching ----------

type MatchFilter = 'all' | 'strong' | 'possible' | 'not_a_fit'

export function Matching() {
  const { t, lang } = useApp()
  const d = useData<{ blueprint: Blueprint | null; fits: Fit[] }>('/matches')
  const [busy, setBusy] = useState<'' | 'bp' | 'match'>('')
  const [drafting, setDrafting] = useState(0)
  const [err, setErr] = useState('')
  const [trBlueprint, setTrBlueprint] = useState<string[] | null>(null)
  const [filter, setFilter] = useState<MatchFilter>('all')

  const buildBp = async () => {
    setBusy('bp'); setErr('')
    try { await api('/blueprint', { body: {} }); await d.reload() } catch (e) { setErr((e as Error).message) } finally { setBusy('') }
  }
  const match = async () => {
    setBusy('match'); setErr('')
    try { d.setData(await api('/matches', { body: {} })) } catch (e) { setErr((e as Error).message) } finally { setBusy('') }
  }
  const draft = async (id: number) => {
    setDrafting(id)
    try { const r = await api('/drafts', { body: { grant_id: id } }); navigate(`/app/apply/${r.id}`) } catch (e) { setErr((e as Error).message); setDrafting(0) }
  }
  const translateBp = async () => {
    const bp = d.data?.blueprint?.data
    if (!bp) return
    const r = await api<{ texts: string[] }>('/translate', { body: { lang, texts: [bp.one_liner, bp.problem, bp.offering, bp.customers, bp.revenue_model] } })
    setTrBlueprint(r.texts)
  }
  useEffect(() => { setTrBlueprint(null) }, [lang])

  const fits = d.data?.fits ?? []
  const strongFits = fits.filter((f) => f.verdict === 'strong')
  const possibleFits = fits.filter((f) => f.verdict === 'possible')
  const notFits = fits.filter((f) => f.verdict === 'not_a_fit')
  const good = fits.filter((f) => f.verdict !== 'not_a_fit')
  const bad = notFits

  const filteredFits = filter === 'strong'
    ? strongFits
    : filter === 'possible'
    ? possibleFits
    : filter === 'not_a_fit'
    ? notFits
    : fits

  return (
    <>
      <PageHead kicker={t('matching_kicker')} title={t('matching_title')}>
        <button className="btn sm" onClick={buildBp} disabled={!!busy}>{busy === 'bp' ? <Spinner /> : <Icon name="book" size={16} />}{d.data?.blueprint ? t('rebuild') : t('build_blueprint')}</button>
        <button className="btn primary sm" onClick={match} disabled={!!busy}>{busy === 'match' ? <Spinner /> : <Icon name="target" size={16} />}{t('run_match')}</button>
      </PageHead>
      <ErrorLine msg={err} />
      {d.loading && !d.data && <div className="skeleton" style={{ height: 300 }} />}
      <div className="stack lg">
        {d.data?.blueprint ? (
          <details className="sheet plain" open={fits.length === 0}>
            <summary style={{ cursor: 'pointer', fontWeight: 700 }}>{t('blueprint')} · {d.data.blueprint.created_at}</summary>
            <div style={{ marginTop: '0.8rem' }}>
              {lang !== 'en' && !trBlueprint && <button className="btn ghost sm" onClick={translateBp}>{t('lang')} → {lang.toUpperCase()}</button>}
              {trBlueprint && <div className="sheet ruled" style={{ lineHeight: '32px', marginBottom: 10 }}>{trBlueprint.map((x, i) => <p key={i}>{x}</p>)}</div>}
              <Link className="btn sm" style={{ marginBottom: 10 }} to="/app/blueprint"><Icon name="flow" size={15} />{t('flowchart')} →</Link>
              <BlueprintView b={d.data.blueprint} />
            </div>
          </details>
        ) : d.data && <Empty title={t('build_blueprint')} hint={t('financials')} />}
        {busy === 'match' && <div className="sheet"><p className="small muted">{t('thinking')}</p><Spinner /></div>}
        {fits.length > 0 && (
          <div className="seg" role="group" style={{ marginBottom: '0.5rem', flexWrap: 'wrap' }}>
            <button type="button" aria-pressed={filter === 'all'} onClick={() => setFilter('all')}>
              {t('all')} ({fits.length})
            </button>
            <button type="button" aria-pressed={filter === 'strong'} onClick={() => setFilter('strong')}>
              <span style={{ color: 'var(--leaf)', marginRight: 5 }}>●</span>
              {t('strong')} ({strongFits.length})
            </button>
            <button type="button" aria-pressed={filter === 'possible'} onClick={() => setFilter('possible')}>
              <span style={{ color: 'var(--turmeric)', marginRight: 5 }}>●</span>
              {t('possible')} ({possibleFits.length})
            </button>
            <button type="button" aria-pressed={filter === 'not_a_fit'} onClick={() => setFilter('not_a_fit')}>
              <span style={{ color: 'var(--stamp)', marginRight: 5 }}>●</span>
              {t('not_a_fit')} ({notFits.length})
            </button>
          </div>
        )}
        {filter === 'all' ? (
          <>
            {good.map((f) => <FitCard key={f.grant_id} f={f} onDraft={draft} drafting={drafting === f.grant_id} />)}
            {bad.length > 0 && (
              <details>
                <summary className="caps muted" style={{ cursor: 'pointer' }}>{t('why_not')} — {bad.length} {t('not_a_fit')}</summary>
                <div className="stack" style={{ marginTop: '0.8rem' }}>{bad.map((f) => <FitCard key={f.grant_id} f={f} />)}</div>
              </details>
            )}
          </>
        ) : filteredFits.length > 0 ? (
          filteredFits.map((f) => <FitCard key={f.grant_id} f={f} onDraft={draft} drafting={drafting === f.grant_id} />)
        ) : (
          <div className="sheet plain muted small" style={{ textAlign: 'center', padding: '1.5rem' }}>
            0 {t(filter)}
          </div>
        )}
      </div>
    </>
  )
}

// ---------- Applications ----------

type Draft = { id: number; title: string; status: string; updated_at: string; needed_count: number; sections: { heading: string; body: string }[]; documents: { name: string; have: boolean }[]; cover_note: string; grant: { id: number; title: string; source: string; source_url: string; deadline: string | null; fetched_at: string } }

export function Apply() {
  const { t } = useApp()
  const d = useData<Draft[]>('/drafts')
  return (
    <>
      <PageHead kicker={t('apply_kicker')} title={t('apply_title')} />
      {d.loading && <Spinner />}
      {d.data?.length === 0 && <Empty title={t('empty_drafts')} hint={t('empty_drafts_hint')}><Link className="btn primary" to="/app/matching">{t('nav_matching')}</Link></Empty>}
      <div className="grid2">
        {d.data?.map((x) => (
          <Link key={x.id} to={`/app/apply/${x.id}`} className="sheet stack" style={{ textDecoration: 'none', gap: '0.4rem' }}>
            <div className="row between"><span className={`stamp ${x.status === 'ready' ? 'green' : 'yellow'}`}>{x.status}</span><span className="tiny faint mono">{x.updated_at}</span></div>
            <h3>{x.title}</h3>
            <p className="small muted">{x.grant.title}</p>
            {x.needed_count > 0 && <span className="chip yellow" style={{ alignSelf: 'flex-start' }}>{t('gaps', { n: x.needed_count })}</span>}
          </Link>
        ))}
      </div>
    </>
  )
}

export function DraftEditor({ id }: { id: number }) {
  const { t, lang, toast } = useApp()
  const list = useData<Draft[]>('/drafts')
  const d = list.data?.find((x) => x.id === id)
  const [sections, setSections] = useState<Draft['sections'] | null>(null)
  useEffect(() => { if (d) setSections(d.sections) }, [d])
  if (list.loading) return <Spinner />
  if (!d || !sections) return <Empty title="—" />

  const save = async (status?: string) => {
    try { await api(`/drafts/${id}`, { method: 'PUT', body: { sections, status } }); toast(t('saved')); list.reload() } catch (e) { toast((e as Error).message) }
  }
  const docx = async () => {
    const { AlignmentType, Document, HeadingLevel, Packer, Paragraph, TextRun } = await import('docx')
    const para = (text: string) => new Paragraph({ children: text.split(/(\[NEEDED:[^\]]*\])/g).map((p) => new TextRun({ text: p, highlight: p.startsWith('[NEEDED') ? 'yellow' : undefined })), spacing: { after: 120 } })
    const doc = new Document({
      sections: [{
        children: [
          new Paragraph({ text: d.title, heading: HeadingLevel.TITLE }),
          new Paragraph({ children: [new TextRun({ text: `${d.grant.title} — ${d.grant.source_url.split('#')[0]}`, italics: true })], alignment: AlignmentType.LEFT }),
          new Paragraph({ text: 'Cover note', heading: HeadingLevel.HEADING_2 }),
          para(d.cover_note),
          ...sections.flatMap((s) => [new Paragraph({ text: s.heading, heading: HeadingLevel.HEADING_2 }), ...s.body.split('\n').filter(Boolean).map(para)]),
          new Paragraph({ text: 'Documents checklist', heading: HeadingLevel.HEADING_2 }),
          ...d.documents.map((x) => new Paragraph({ text: `${x.have ? '☑' : '☐'} ${x.name}` })),
          new Paragraph({ children: [new TextRun({ text: 'Prepared with Finro. Verify every detail and the deadline on the official scheme website before submitting.', italics: true, size: 18 })] }),
        ],
      }],
    })
    const blob = await Packer.toBlob(doc)
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = `${d.title.replace(/[^\w]+/g, '-').slice(0, 60)}.docx`
    a.click()
  }

  return (
    <>
      <div className="row between wrap no-print" style={{ marginBottom: '1rem' }}>
        <Link className="btn ghost sm" to="/app/apply">← {t('apply_title')}</Link>
        <div className="row wrap">
          <button className="btn sm" onClick={() => save()}>{t('save')}</button>
          <button className="btn sm" onClick={docx}><Icon name="download" size={16} />{t('export_docx')}</button>
          <button className="btn sm" onClick={() => print()}><Icon name="printer" size={16} />{t('export_pdf')}</button>
          <button className="btn primary sm" onClick={() => save('ready')}><Icon name="check" size={16} />{t('mark_ready')}</button>
        </div>
      </div>
      <div className="grid2" style={{ gridTemplateColumns: 'minmax(0, 2fr) minmax(240px, 1fr)', alignItems: 'start' }}>
        <article className="draft-doc">
          <div className="row between wrap"><span className="caps faint">{d.grant.source}</span><span className={`stamp ${d.status === 'ready' ? 'green' : 'yellow'}`}>{d.status}</span></div>
          <h2 style={{ marginTop: 6 }}>{d.title}</h2>
          <p className="small muted">{d.grant.title} · <a href={d.grant.source_url.split('#')[0]} target="_blank" rel="noopener noreferrer">{t('view_source')} ↗</a></p>
          <h3>{t('cover_note')}</h3>
          <Md text={d.cover_note} />
          {sections.map((s, i) => (
            <section key={i}>
              <h3>{s.heading}</h3>
              <div contentEditable suppressContentEditableWarning onBlur={(e) => { const next = [...sections]; next[i] = { ...s, body: e.currentTarget.innerText }; setSections(next) }}>
                <Md text={s.body} />
              </div>
            </section>
          ))}
        </article>
        <aside className="stack no-print">
          <div className="sheet">
            <h3>{t('docs_checklist')}</h3>
            <ul style={{ listStyle: 'none', padding: 0, margin: '0.5rem 0 0' }}>
              {d.documents.map((x) => <li key={x.name} className="row" style={{ gap: 6, padding: '3px 0' }}><span className={`chip ${x.have ? 'green' : 'red'}`}>{x.have ? t('have') : t('need')}</span>{x.name}</li>)}
            </ul>
          </div>
          <div className="sheet small">
            <span className="chip yellow">{t('gaps', { n: d.needed_count })}</span>
            <p className="muted" style={{ marginTop: 8 }}>{d.grant.deadline ? `${t('deadline')}: ${d.grant.deadline}` : t('no_deadline')}</p>
            <p className="tiny faint">{t('fetched', { t: ago(d.grant.fetched_at, lang) })}</p>
          </div>
        </aside>
      </div>
    </>
  )
}
