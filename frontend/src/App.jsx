import { useEffect, useRef, useState, useCallback } from 'react'
import { motion } from 'framer-motion'
import './index.css'

/* ─── Background floats ─────────────────────────────── */
function Floats() {
  return (
    <div className="floats" aria-hidden="true">
      <div className="float-orb" />
      <div className="float-orb" />
      <div className="float-orb" />
      <div className="float-box" />
      <div className="float-box" />
      <div className="float-box" />
      <div className="float-box" />
      <div className="float-dot" />
      <div className="float-dot" />
      <div className="float-dot" />
      <div className="float-dot" />
      <div className="float-dot" />
      <div className="float-dot" />
      <div className="float-cross" />
      <div className="float-cross" />
      <div className="float-cross" />
    </div>
  )
}

/* ─── SVG Icons ─────────────────────────────────────── */
const Icon = ({ name, size = 16, ...p }) => {
  const paths = {
    scan:   <><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/><path d="M11 8v6M8 11h6"/></>,
    docker: <><path d="M22 12.5c0 1-1 2-2 2H4c-1 0-2-1-2-2V9c0-1 1-2 2-2h4V5c0-1 1-2 2-2h4c1 0 2 1 2 2v2h4c1 0 2 1 2 2z"/><path d="M6 9h2M10 9h2M14 9h2M10 5h2"/></>,
    globe:  <><circle cx="12" cy="12" r="10"/><path d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10A15.3 15.3 0 0 1 12 2z"/></>,
    git:    <><circle cx="18" cy="18" r="3"/><circle cx="6"  cy="6"  r="3"/><path d="M13 6h3a2 2 0 0 1 2 2v7"/><circle cx="6" cy="18" r="3"/><path d="M6 9v6"/></>,
    lock:   <><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></>,
    pulse:  <><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></>,
    arrow:  <><path d="M5 12h14M12 5l7 7-7 7"/></>,
    plus:   <><path d="M12 5v14M5 12h14"/></>,
    check:  <><polyline points="20 6 9 17 4 12"/></>,
    term:   <><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></>,
  }
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" {...p}>
      {paths[name]}
    </svg>
  )
}

/* ─── Elegant hero shapes ───────────────────────────── */
function ElegantShape({ delay = 0, width = 400, height = 100, rotate = 0, gradient, style: pos }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: -150, rotate: rotate - 15 }}
      animate={{ opacity: 1, y: 0, rotate }}
      transition={{ duration: 2.4, delay, ease: [0.23, 0.86, 0.39, 0.96], opacity: { duration: 1.2 } }}
      style={{ position: 'absolute', ...pos }}
    >
      <motion.div
        animate={{ y: [0, 15, 0] }}
        transition={{ duration: 12, repeat: Infinity, ease: 'easeInOut' }}
        style={{ width, height, position: 'relative' }}
      >
        <div style={{
          position: 'absolute', inset: 0, borderRadius: '9999px',
          background: gradient,
          backdropFilter: 'blur(2px)',
          border: '1px solid rgba(197,247,0,0.10)',
          boxShadow: '0 8px 32px 0 rgba(197,247,0,0.05)',
        }} />
      </motion.div>
    </motion.div>
  )
}

/* ─── Cursor glow ───────────────────────────────────── */
function useCursorGlow(ref) {
  useEffect(() => {
    const el = ref.current
    if (!el) return
    const h = (e) => {
      const r = el.getBoundingClientRect()
      el.style.setProperty('--mx', `${e.clientX - r.left}px`)
      el.style.setProperty('--my', `${e.clientY - r.top}px`)
    }
    el.addEventListener('mousemove', h, { passive: true })
    return () => el.removeEventListener('mousemove', h)
  }, [ref])
}

/* ─── Smart nav ─────────────────────────────────────── */
function useSmartNav() {
  useEffect(() => {
    const nav = document.querySelector('nav')
    if (!nav) return
    const h = () => nav.classList.toggle('solid', window.scrollY > 40)
    window.addEventListener('scroll', h, { passive: true })
    return () => window.removeEventListener('scroll', h)
  }, [])
}

/* ─── Text scramble ─────────────────────────────────── */
const CHARS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
function scramble(el, final) {
  let frame = 0
  const total = 18
  const tick = () => {
    el.textContent = final.split('').map((c, i) => {
      if (c === ' ') return ' '
      if (frame / total > i / final.length) return c
      return CHARS[Math.floor(Math.random() * CHARS.length)]
    }).join('')
    frame++
    if (frame <= total) requestAnimationFrame(tick)
    else el.textContent = final
  }
  requestAnimationFrame(tick)
}

function useScrambleReveal() {
  useEffect(() => {
    const els = document.querySelectorAll('[data-scramble]')
    const obs = new IntersectionObserver(entries => {
      entries.forEach(e => {
        if (e.isIntersecting) {
          scramble(e.target, e.target.dataset.scramble)
          obs.unobserve(e.target)
        }
      })
    }, { threshold: 0.4 })
    els.forEach(el => obs.observe(el))
    return () => obs.disconnect()
  }, [])
}

/* ─── Terminal ──────────────────────────────────────── */
const AUTO_SEQ = [
  { k: 'cmd',     t: '$ ship deploy' },
  { k: 'blank' },
  { k: 'muted',   t: 'Analyzing project...' },
  { k: 'blank' },
  { k: 'ok',      t: '✓ Django detected' },
  { k: 'ok',      t: '✓ PostgreSQL found' },
  { k: 'ok',      t: '✓ Redis detected' },
  { k: 'blank' },
  { k: 'label',   t: 'Describe your deployment' },
  { k: 'user',    t: 'Deploy with PostgreSQL & Redis. HTTPS. Auto-deploy on push to main.' },
  { k: 'blank' },
  { k: 'spin',    t: 'Generating deployment plan' },
  { k: 'blank' },
  { k: 'ok',      t: '✓ Dockerfile generated' },
  { k: 'ok',      t: '✓ Docker Compose ready' },
  { k: 'ok',      t: '✓ Nginx configured' },
  { k: 'ok',      t: '✓ GitHub Actions created' },
  { k: 'ok',      t: "✓ SSL via Let's Encrypt" },
  { k: 'blank' },
  { k: 'url',     t: '🚀  https://myapp.example.com' },
]
const FRAMES = ['⠋','⠙','⠹','⠸','⠼','⠴','⠦','⠧','⠇','⠏']

function Terminal() {
  const [lines,  setLines]  = useState([])
  const [typing, setTyping] = useState(null)
  const [spinner, setSpinner] = useState(null)
  const [spinF,  setSpinF]  = useState(0)
  const [mode,   setMode]   = useState('auto')   // 'auto' | 'input' | 'responding'
  const [userInput, setUserInput] = useState('')
  const inputRef = useRef(null)
  const bodyRef  = useRef(null)
  const timerRef = useRef(null)

  const scroll = () => { if (bodyRef.current) bodyRef.current.scrollTop = 9999 }

  // Spinner tick
  useEffect(() => {
    if (!spinner) return
    const t = setInterval(() => setSpinF(i => i + 1), 75)
    return () => clearInterval(t)
  }, [spinner])

  useEffect(() => { scroll() }, [lines, typing, spinner, userInput])

  const run = useCallback(() => {
    let step = 0
    setLines([]); setTyping(null); setSpinner(null)

    const next = () => {
      if (step >= AUTO_SEQ.length) { timerRef.current = setTimeout(run, 3000); return }
      const item = AUTO_SEQ[step++]

      if (item.k === 'blank') {
        setLines(l => [...l, { k: 'blank', id: Math.random() }])
        timerRef.current = setTimeout(next, 100); return
      }
      if (item.k === 'spin') {
        setSpinner(item.t)
        timerRef.current = setTimeout(() => {
          setSpinner(null)
          setLines(l => [...l, { k: 'ok', t: `✓ ${item.t} ready`, id: Math.random() }])
          timerRef.current = setTimeout(next, 150)
        }, 1400); return
      }
      if (item.k === 'cmd' || item.k === 'user') {
        let i = 0
        setTyping({ k: item.k, t: item.t, r: '' })
        const tick = () => {
          i++
          setTyping({ k: item.k, t: item.t, r: item.t.slice(0, i) })
          if (i < item.t.length) {
            timerRef.current = setTimeout(tick, item.k === 'cmd' ? 52 : 15)
          } else {
            timerRef.current = setTimeout(() => {
              setTyping(null)
              setLines(l => [...l, { ...item, id: Math.random() }])
              timerRef.current = setTimeout(next, item.k === 'cmd' ? 380 : 280)
            }, 220)
          }
        }
        timerRef.current = setTimeout(tick, item.k === 'cmd' ? 500 : 60); return
      }
      setLines(l => [...l, { ...item, id: Math.random() }])
      timerRef.current = setTimeout(next, { ok: 140, muted: 280, label: 180, url: 0 }[item.k] ?? 180)
    }
    timerRef.current = setTimeout(next, 300)
  }, [])

  useEffect(() => {
    run()
    return () => clearTimeout(timerRef.current)
  }, [run])

  // Interactive mode
  const enterInteractive = () => {
    if (mode !== 'auto') return
    clearTimeout(timerRef.current)
    setMode('input')
    setLines([{ k: 'cmd', t: '$ ship deploy', id: 1 }, { k: 'blank', id: 2 }, { k: 'label', t: 'Describe your deployment', id: 3 }])
    setTyping(null); setSpinner(null); setUserInput('')
    setTimeout(() => inputRef.current?.focus(), 50)
  }

  const handleKey = (e) => {
    if (mode !== 'input') return
    e.preventDefault()
    if (e.key === 'Enter' && userInput.trim()) {
      const desc = userInput
      setLines(l => [...l, { k: 'user', t: desc, id: Math.random() }, { k: 'blank', id: Math.random() }])
      setUserInput('')
      setMode('responding')

      const responses = [
        { k: 'spin', t: 'Generating deployment plan' },
        { k: 'ok',   t: '✓ Dockerfile generated' },
        { k: 'ok',   t: '✓ Docker Compose ready' },
        { k: 'ok',   t: '✓ Nginx configured' },
        { k: 'ok',   t: '✓ GitHub Actions created' },
        { k: 'ok',   t: '✓ SSL enabled' },
        { k: 'blank' },
        { k: 'url',  t: '🚀  https://myapp.example.com' },
      ]
      let delay = 300
      responses.forEach((r, i) => {
        timerRef.current = setTimeout(() => {
          if (r.k === 'spin') {
            setSpinner(r.t)
            setTimeout(() => {
              setSpinner(null)
              setLines(l => [...l, { k: 'ok', t: `✓ ${r.t} ready`, id: Math.random() }])
            }, 1200)
          } else {
            setLines(l => [...l, { ...r, id: Math.random() }])
          }
          if (i === responses.length - 1) {
            setTimeout(() => { setMode('done') }, 500)
          }
        }, delay)
        delay += r.k === 'spin' ? 1600 : r.k === 'blank' ? 200 : 160
      })
    } else if (e.key === 'Backspace') {
      setUserInput(v => v.slice(0, -1))
    } else if (e.key.length === 1) {
      setUserInput(v => v + e.key)
    }
  }

  const lineColor = (k) => ({ cmd: 't-text-cmd', muted: 't-text-muted', ok: 't-text-success', label: 't-text-label', user: 't-text-user', url: 't-text-url' }[k] || 't-text-muted')

  return (
    <div className="terminal-wrap" onClick={enterInteractive}>
      <div className="terminal-bar">
        <span className="t-dot t-red" /><span className="t-dot t-yellow" /><span className="t-dot t-green" />
        <span className="t-title">ship — deploy</span>
      </div>
      <div className="terminal-body" ref={bodyRef} onKeyDown={handleKey} tabIndex={0} style={{ outline: 'none' }}>
        {lines.map(l =>
          l.k === 'blank' ? <div key={l.id} className="t-blank" /> :
          l.k === 'user' ? <div key={l.id} className="t-line"><span className="t-prompt-char">›</span><span className={lineColor(l.k)}>{l.t}</span></div> :
          <div key={l.id} className="t-line"><span className={lineColor(l.k)}>{l.t}</span></div>
        )}
        {spinner && (
          <div className="t-line">
            <span className="t-text-spinner">{FRAMES[spinF % FRAMES.length]}</span>
            <span className="t-text-muted"> {spinner}...</span>
          </div>
        )}
        {typing && (
          <div className="t-line">
            {typing.k === 'user' && <span className="t-prompt-char">›</span>}
            <span className={typing.k === 'cmd' ? 't-text-cmd' : 't-text-user'}>{typing.r}</span>
            <span className="t-cursor" />
          </div>
        )}
        {mode === 'input' && (
          <div className="t-line">
            <span className="t-prompt-char">›</span>
            <span className="t-text-user">{userInput}</span>
            <span className="t-cursor" />
          </div>
        )}
        {mode === 'auto' && !typing && !spinner && lines.length === 0 && <span className="t-cursor" />}
        {mode === 'auto' && !typing && !spinner && lines.length > 0 && <span className="t-cursor" style={{ opacity: 0 }} />}
        {mode === 'input' && (
          <div className="t-hint">press enter to generate plan</div>
        )}
        {mode === 'done' && (
          <div className="t-hint" style={{ cursor: 'pointer' }} onClick={(e) => { e.stopPropagation(); setMode('auto'); run() }}>
            click to restart demo
          </div>
        )}
      </div>
    </div>
  )
}

/* ─── Marquee ───────────────────────────────────────── */
const TECH = ['Docker','Docker Compose','Nginx','PostgreSQL','Redis','GitHub Actions','Let\'s Encrypt','SSH','Certbot','Ubuntu VPS']
function Marquee() {
  const items = [...TECH, ...TECH]
  return (
    <div className="marquee-strip">
      <div className="marquee-track">
        {items.map((t, i) => (
          <span className="marquee-item" key={i}>
            <Icon name="check" size={12} />
            {t}
            {i < items.length - 1 && <span className="marquee-dot" />}
          </span>
        ))}
      </div>
    </div>
  )
}

/* ─── Features ──────────────────────────────────────── */
const FEATURES = [
  { icon: 'scan',   name: 'Project detection',   desc: 'Auto-detects your language, framework, port, and dependencies. No config files needed.', detail: 'Supports Go, Python, Node.js, Ruby, Java, Rust, and PHP. Detects Django, FastAPI, Express, Next.js, Rails, Gin, and more out of the box.' },
  { icon: 'docker', name: 'Docker generation',    desc: 'Multi-stage Dockerfiles and Docker Compose with health checks, non-root users, and restart policies.', detail: 'Generates production-ready configurations tailored to your stack. PostgreSQL, Redis, and worker services included automatically when detected.' },
  { icon: 'globe',  name: 'Nginx + HTTPS',        desc: 'Nginx runs as a Docker container — no system permissions required. Certbot handles SSL certificates.', detail: 'Your nginx.conf is mounted from your app directory. Let\'s Encrypt certificates are provisioned automatically on first deploy.' },
  { icon: 'git',    name: 'GitHub Actions CI/CD', desc: 'Auto-deploy on push to main. Full workflow: build, push, SSH into server, restart containers.', detail: 'The generated workflow file goes straight into .github/workflows/deploy.yml. Secrets mapping is included in the deployment plan.' },
  { icon: 'lock',   name: 'Secure by design',     desc: 'SSH keys never leave your machine. All server operations run locally through the CLI.', detail: 'Only project metadata is sent to your configured AI provider. No credentials, no server access, no Ship cloud.' },
  { icon: 'pulse',  name: 'Built-in diagnostics', desc: 'ship doctor checks SSH, Docker, AI key validity, and domain DNS in one command.', detail: 'Identifies the exact failure point with actionable messages. No more digging through logs to find out why a deploy failed.' },
]

function Features() {
  const [open, setOpen] = useState(null)
  return (
    <section id="features" className="content-panel">
      <div className="wrap">
        <div className="features-header">
          <div>
            <span className="eyebrow">Features</span>
            <h2 data-scramble="Everything generated. Nothing configured.">Everything generated. Nothing configured.</h2>
          </div>
        </div>
        <div className="feature-list">
          {FEATURES.map((f, i) => (
            <div key={i}>
              <div
                className={`feature-row${open === i ? ' open' : ''}`}
                onClick={() => setOpen(open === i ? null : i)}
              >
                <div className="feature-name">
                  <Icon name={f.icon} size={16} />
                  {f.name}
                </div>
                <div className="feature-desc">{f.desc}</div>
                <div className={`feature-arrow${open === i ? ' open' : ''}`}>
                  <Icon name="plus" size={16} />
                </div>
              </div>
              <div className={`feature-expand${open === i ? ' open' : ''}`}>
                <div className="feature-expand-inner">{f.detail}</div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

/* ─── How it works ──────────────────────────────────── */
const FLOW_NODES = [
  { icon: 'scan',   label: 'Detect',   lines: ['$ ship init', '', '✓ Go project detected', '✓ Gin framework', '✓ Port 8080', '✓ PostgreSQL found'] },
  { icon: 'pulse',  label: 'Describe', lines: ['› Deploy with HTTPS,', '  PostgreSQL, CI/CD', '  on push to main.'] },
  { icon: 'docker', label: 'Generate', lines: ['✓ Dockerfile', '✓ docker-compose.yml', '✓ nginx.conf', '✓ .github/workflows/'] },
  { icon: 'globe',  label: 'Deploy',   lines: ['✓ Files uploaded', '✓ Containers started', '✓ SSL issued', '', '🚀  https://myapp.com'] },
]

function HowItWorks() {
  const [active, setActive] = useState(0)
  const [visLines, setVisLines] = useState([])

  useEffect(() => {
    setVisLines([])
    FLOW_NODES[active].lines.forEach((l, i) => {
      setTimeout(() => setVisLines(v => [...v, l]), i * 80)
    })
  }, [active])

  return (
    <section id="how" className="content-panel">
      <div className="wrap">
        <span className="eyebrow">How it works</span>
        <h2 data-scramble="Three commands. One deployed app.">Three commands. One deployed app.</h2>

        <div className="flow-wrap">
          {FLOW_NODES.map((n, i) => (
            <div key={i} className={`flow-node${active === i ? ' active' : ''}`} onClick={() => setActive(i)}>
              <div className="flow-icon"><Icon name={n.icon} size={18} /></div>
              <div className="flow-label">{n.label}</div>
              {i < FLOW_NODES.length - 1 && (
                <span className="flow-arrow-icon"><Icon name="arrow" size={12} /></span>
              )}
            </div>
          ))}
        </div>

        <div className="flow-preview-box">
          {visLines.map((l, i) => (
            <div key={i} className={`preview-line${l.startsWith('✓') || l.startsWith('🚀') ? ' accent' : l.startsWith('$') ? ' cmd' : ''}`}>
              {l}
            </div>
          ))}
          {visLines.length < FLOW_NODES[active].lines.length && <span className="t-cursor" />}
        </div>
      </div>
    </section>
  )
}

/* ─── Stats ─────────────────────────────────────────── */
function Stats() {
  const STATS = [
    { n: '7',   unit: '',   label: 'Languages detected' },
    { n: '3',   unit: '',   label: 'AI providers' },
    { n: '<5',  unit: 'min',label: 'First deploy' },
    { n: '100%', unit: '',  label: 'Your infrastructure' },
  ]
  return (
    <div className="wrap">
      <div className="stats-row">
        {STATS.map((s, i) => (
          <div className="stat-item" key={i}>
            <div className="stat-n">{s.n}<span style={{ fontSize: '60%', opacity: 0.7, marginLeft: 2 }}>{s.unit}</span></div>
            <div className="stat-label">{s.label}</div>
          </div>
        ))}
      </div>
    </div>
  )
}

/* ─── Copy btn ──────────────────────────────────────── */
function CopyBtn({ text, label }) {
  const [done, setDone] = useState(false)
  const copy = () => {
    navigator.clipboard.writeText(text)
    setDone(true)
    setTimeout(() => setDone(false), 1800)
  }
  return (
    <button className={`copy-btn${done ? ' copied' : ''}`} onClick={copy}>
      {done ? '✓ copied' : label || 'copy'}
    </button>
  )
}

/* ─── FAQ ───────────────────────────────────────────── */
const FAQS = [
  { q: 'Does Ship host my app?',          a: 'No. Ship deploys to your own VPS. You keep full control of your infrastructure, provider, and data.' },
  { q: 'Does it require Docker?',         a: "Docker is the default runtime, but Ship generates the Dockerfile for you — you don't need to write it yourself." },
  { q: 'Does Ship store my SSH keys?',    a: 'Never. Keys are read from your local machine and used directly. Nothing credential-related is transmitted anywhere.' },
  { q: 'Which AI providers are supported?', a: 'Claude (Anthropic), OpenAI GPT-4o, and Google Gemini. Bring your own API key — only project metadata is sent to generate the plan.' },
]
function FAQ() {
  const [open, setOpen] = useState(null)
  return (
    <section id="faq">
      <div className="wrap">
        <span className="eyebrow">FAQ</span>
        <h2 data-scramble="Common questions.">Common questions.</h2>
        <div className="faq-list">
          {FAQS.map((f, i) => (
            <div className="faq-item" key={i}>
              <div className="faq-q" onClick={() => setOpen(open === i ? null : i)}>
                <h3>{f.q}</h3>
                <span className={`faq-icon${open === i ? ' open' : ''}`}><Icon name="plus" size={16} /></span>
              </div>
              <div className={`faq-a${open === i ? ' open' : ''}`}><p>{f.a}</p></div>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

/* ─── App ───────────────────────────────────────────── */
export default function App() {
  const heroRef = useRef(null)
  useCursorGlow(heroRef)
  useSmartNav()
  useScrambleReveal()

  return (
    <>
      <Floats />
      <div style={{ position: 'relative', zIndex: 1 }}>
      <nav className="up">
        <div className="dock">
          <a href="#" className="dock-logo"><span className="dollar">$</span>&nbsp;ship</a>
          <div className="dock-sep" />
          <a href="#features" className="dock-item"><Icon name="scan" size={13} />Features</a>
          <a href="#how" className="dock-item"><Icon name="term" size={13} />How it works</a>
          <a href="#roadmap" className="dock-item"><Icon name="pulse" size={13} />Roadmap</a>
          <a href="https://github.com/Koded0214h/ship-it" className="dock-item"><Icon name="git" size={13} />GitHub</a>
          <div className="dock-sep" />
          <a href="#install" className="dock-cta">Install <Icon name="arrow" size={12} /></a>
        </div>
      </nav>

      {/* ── Hero ── */}
      <section id="hero" ref={heroRef}>
        {/* floating pill shapes */}
        <ElegantShape delay={0.3} width={560} height={130} rotate={12}
          gradient="linear-gradient(to right, rgba(197,247,0,0.10), transparent)"
          style={{ left: '-6%', top: '16%' }} />
        <ElegantShape delay={0.5} width={420} height={100} rotate={-14}
          gradient="linear-gradient(to right, rgba(197,247,0,0.07), transparent)"
          style={{ right: '-3%', top: '60%' }} />
        <ElegantShape delay={0.4} width={260} height={70} rotate={-7}
          gradient="linear-gradient(to right, rgba(255,255,255,0.04), transparent)"
          style={{ left: '4%', bottom: '8%' }} />
        <ElegantShape delay={0.65} width={180} height={50} rotate={22}
          gradient="linear-gradient(to right, rgba(197,247,0,0.08), transparent)"
          style={{ right: '10%', top: '6%' }} />
        <ElegantShape delay={0.75} width={120} height={36} rotate={-28}
          gradient="linear-gradient(to right, rgba(255,255,255,0.03), transparent)"
          style={{ left: '20%', top: '5%' }} />

        <div className="hero-center">
          <motion.h1
            className="hero-headline"
            initial={{ opacity: 0, y: 40 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 1, delay: 0.35, ease: [0.25, 0.4, 0.25, 1] }}
          >
            SHIP <span className="accent">IT.</span>
          </motion.h1>
          <motion.p
            className="hero-sub"
            initial={{ opacity: 0, y: 28 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 1, delay: 0.52, ease: [0.25, 0.4, 0.25, 1] }}
          >
            Describe your deployment. Ship generates the infrastructure and runs it on your own server.
          </motion.p>
          <motion.div
            className="hero-ctas"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 1, delay: 0.68, ease: [0.25, 0.4, 0.25, 1] }}
          >
            <a href="#install" className="btn btn-primary">Get Ship <Icon name="arrow" size={14} /></a>
            <a href="https://github.com/Koded0214h/ship-it" className="btn btn-outline">GitHub</a>
          </motion.div>
          <motion.div
            className="hero-terminal"
            initial={{ opacity: 0, y: 36 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 1.1, delay: 0.82, ease: [0.25, 0.4, 0.25, 1] }}
          >
            <Terminal />
            <p style={{ fontSize: '0.7rem', color: 'var(--text-3)', marginTop: 10, fontFamily: 'var(--mono)', textAlign: 'center' }}>
              click the terminal to try it yourself
            </p>
          </motion.div>
        </div>
      </section>

      <Marquee />
      <Stats />
      <Features />
      <HowItWorks />

      {/* ── Install ── */}
      <section id="install">
        <div className="wrap">
          <span className="eyebrow">Get started</span>
          <div className="install-grid">
            <div>
              <h2 data-scramble="Start deploying in minutes.">Start deploying in minutes.</h2>
              <p style={{ marginTop: 16 }}>
                Install Ship, connect your server, and deploy your first app.
                No cloud account, no dashboard, no lock-in.
              </p>
              <div style={{ display: 'flex', gap: 10, marginTop: 28 }}>
                <a href="https://github.com/Koded0214h/ship-it" className="btn btn-primary">View on GitHub <Icon name="arrow" size={14} /></a>
              </div>
            </div>
            <div className="install-cmds">
              {[
                { label: 'Homebrew', cmd: 'brew install ship' },
                { label: 'npm',      cmd: 'npm install -g ship-cli' },
                { label: 'curl',     cmd: 'curl -fsSL shipcli.dev/install.sh | bash' },
              ].map(({ label, cmd }) => (
                <div key={label}>
                  <div style={{ fontFamily: 'var(--mono)', fontSize: '0.68rem', color: 'var(--text-3)', marginBottom: 5, letterSpacing: '0.08em', textTransform: 'uppercase' }}>{label}</div>
                  <div className="install-cmd">
                    <span className="cmd">{cmd}</span>
                    <CopyBtn text={cmd} />
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* ── Roadmap ── */}
      <section id="roadmap" className="content-panel">
        <div className="wrap">
          <span className="eyebrow">Roadmap</span>
          <h2 data-scramble="Where Ship is headed.">Where Ship is headed.</h2>
          <div className="roadmap-grid">
            <div className="rm-col">
              <div className="rm-label done">Shipped ↗</div>
              <div className="rm-list">
                {['Project detection','SSH connections','Dockerfile generation','Docker Compose generation','GitHub Actions CI/CD','Nginx configuration','HTTPS via Let\'s Encrypt','Live log streaming','ship doctor diagnostics'].map(t => (
                  <div className="rm-item" key={t}>
                    <Icon name="check" size={14} className="rm-check" style={{ color: 'var(--accent)', flexShrink: 0 }} />
                    {t}
                  </div>
                ))}
              </div>
            </div>
            <div className="rm-col">
              <div className="rm-label soon">Coming soon</div>
              <div className="rm-list">
                {['Rollbacks','Deployment history','Interactive plan editing','Environment variable management','Monitoring integrations','Automatic backups','Multi-server deployments','Background workers & cron','Plugin system'].map(t => (
                  <div className="rm-item" key={t} style={{ color: 'var(--text-3)' }}>
                    <span style={{ color: 'var(--text-3)', flexShrink: 0, width: 14, display: 'inline-flex', alignItems: 'center', justifyContent: 'center' }}>—</span>
                    {t}
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </section>

      <FAQ />

      <footer>
        <div className="footer-inner">
          <div className="footer-logo"><span>$</span> ship v0.1.0 · MIT License</div>
          <div className="footer-links">
            <a href="https://github.com/Koded0214h/ship-it">GitHub</a>
            <a href="#roadmap">Roadmap</a>
            <a href="#faq">FAQ</a>
          </div>
          <div className="footer-copy">Built by KodedLabs</div>
        </div>
      </footer>
      </div>
    </>
  )
}
