// Pre-builds AI artifacts for the demo personas on a deployed Finro so judges see them instantly.
// Usage: node scripts/prime-demo.mjs [baseURL]
const BASE = process.argv[2] || 'https://munshi-ochre.vercel.app'
const personas = [['Lakshmi', '9000000001', '1111'], ['Ramesh', '9000000002', '2222'], ['Mentor', '9000000000', '0000']]

async function session(phone, pin) {
  const r = await fetch(BASE + '/api/auth/login', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ phone, pin }) })
  if (!r.ok) throw new Error('login failed ' + r.status)
  const cookie = r.headers.get('set-cookie').split(';')[0]
  return async (method, path, body) => {
    const t0 = Date.now()
    const res = await fetch(BASE + '/api' + path, { method, headers: { 'Content-Type': 'application/json', cookie }, body: body ? JSON.stringify(body) : undefined })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(`${path} ${res.status} ${data.error ?? ''}`)
    return [data, Date.now() - t0]
  }
}

await Promise.all(personas.map(async ([name, phone, pin]) => {
  const log = (...a) => console.log(`[${name}]`, ...a)
  try {
    const call = await session(phone, pin)
    const [adv, ms1] = await call('POST', '/stock/advice')
    log(`advice ${ms1}ms: "${adv.headline}" more=[${adv.buy_more.map((b) => `${b.item} ${b.qty}${b.unit}`).join(', ')}] less=[${adv.buy_less.map((b) => b.item).join(', ')}]`)
    const [bp, ms2] = await call('POST', '/blueprint', {})
    log(`blueprint #${bp.id} ${ms2}ms: ${bp.data.one_liner} | vision: ${bp.data.vision}`)
    const [ill, ms3] = await call('POST', `/blueprint/${bp.id}/illustrate`)
    log(`images ${ms3}ms: ${ill.slots.map((s) => `${s.slot}=${s.ok ? 'ok ' + s.model : 'FAIL ' + s.error}`).join(', ')}`)
  } catch (e) {
    log('ERROR', e.message)
  }
}))
