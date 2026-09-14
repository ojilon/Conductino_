/**
 * BrowserView placeholder.
 *
 * This is the rendering boundary for web content. Today it draws two
 * realistic mock pages (from URL-keyed data), a mock search-results
 * page, a new-tab page and a generic fallback — all of which react to
 * real tab navigation (links call onNavigate → NavigationState updates).
 *
 * FUTURE INTEGRATION: replace `MockWebPage` with a component that
 * mounts the real browser engine (Wails webview / Go WebView) and
 * forward `tab.nav.currentUrl` into it. See docs/architecture.md
 * §"Browser integration boundary" — nothing outside this file changes.
 */

import { Icon, type IconName } from "../../components/icons";

export const MOCK_PAGE_URL_A = "https://www.nature.com/articles/s41467-020-16742-1";
export const MOCK_PAGE_URL_B = "https://www.nature.com/articles/s41467-020-18306-7";

interface PageData {
  wordmark: string;
  redRule?: boolean;
  nav: string[];
  crumb: string[];
  kicker: { label: string; accent: string; extra: string };
  title: string;
  authors: string[];
  citation: { journal: string; rest: string };
  stats: { v: string; l: string }[];
  abstract: string;
  keywords?: string[];
  sections: string[];
  downloads?: { icon: IconName; label: string }[];
  references?: { n: number; text: string }[];
}

const PAGES: Record<string, PageData> = {
  [MOCK_PAGE_URL_A]: {
    wordmark: "nature",
    nav: ["Explore content", "About the journal", "Publish with us"],
    crumb: ["nature", "articles", "article"],
    kicker: { label: "Article", accent: "Open access", extra: "Published: 02 March 2020" },
    title: "Mechanism of ATP production during chemiosmosis",
    authors: ["Daniel R. Mitchell", "Elena S. García", "James T. Fraser"],
    citation: { journal: "Nature", rest: " 588, 374–381 (2020)" },
    stats: [
      { v: "12k", l: "Accesses" },
      { v: "423", l: "Citations" },
      { v: "32", l: "Altmetric" },
    ],
    abstract:
      "Chemiosmosis couples the electron transport chain to ATP synthesis by exploiting the proton gradient across the inner mitochondrial membrane. Here we provide a detailed mechanistic analysis of how the proton motive force drives ATP synthase activity, integrating structural, biochemical and kinetic data.",
    keywords: ["chemiosmosis", "ATP synthase", "proton gradient", "membrane potential"],
    sections: ["Abstract", "Introduction", "Results", "Discussion", "Methods", "References"],
    downloads: [
      { icon: "fileText", label: "PDF (2.4 MB)" },
      { icon: "file", label: "Supplementary information" },
      { icon: "file", label: "Figures (6)" },
      { icon: "copy", label: "Citation export" },
    ],
    references: [
      { n: 1, text: "Mitchell, P. Chemiosmotic coupling in oxidative and photophosphorylation. Biol. Rev. 41, 445–502 (1966)." },
    ],
  },
  [MOCK_PAGE_URL_B]: {
    wordmark: "nature communications",
    redRule: true,
    nav: ["Explore content", "About the journal", "Publish with us"],
    crumb: ["nature", "nature communications", "articles", "article"],
    kicker: { label: "Article", accent: "Open access", extra: "Published: 14 September 2020" },
    title: "Proton gradients and ATP synthase",
    authors: ["Elena García-Martín", "Daniel J. Müller", "Robert E. Blankenship"],
    citation: { journal: "Nature Communications", rest: " 11, 4567 (2020)" },
    stats: [
      { v: "12k", l: "Accesses" },
      { v: "487", l: "Citations" },
      { v: "21", l: "Altmetric" },
    ],
    abstract:
      "The generation and utilization of proton gradients across biological membranes is central to cellular energy conversion. In chloroplasts, the light-driven electron transport chain establishes a proton motive force that powers ATP synthase, linking photochemistry to the synthesis of ATP. Here we review the molecular mechanisms of proton translocation, the structure and function of chloroplast ATP synthase, and how the proton gradient is regulated during chemiosmosis.",
    keywords: ["chemiosmosis", "ATP synthase", "proton gradient", "photosynthesis", "chloroplast"],
    sections: ["Abstract", "Introduction", "Results", "Discussion", "Methods", "References"],
    downloads: [
      { icon: "fileText", label: "PDF (2.4 MB)" },
      { icon: "file", label: "Supplementary information" },
      { icon: "file", label: "Figures (6)" },
      { icon: "copy", label: "Citation export" },
    ],
    references: [
      { n: 1, text: "Mitchell, P. Chemiosmotic coupling in oxidative and photophosphorylation. Biol. Rev. 41, 445–502 (1966)." },
    ],
  },
};

/* small in-page link (mock page links drive real tab navigation) */
function PageLink({
  children,
  onClick,
  className = "",
}: {
  children: React.ReactNode;
  onClick: () => void;
  className?: string;
}) {
  return (
    <a
      href="#"
      onClick={(e) => {
        e.preventDefault();
        onClick();
      }}
      className={`text-[#1a56c4] hover:underline ${className}`}
    >
      {children}
    </a>
  );
}

function ArticlePage({ url, data, onNavigate }: { url: string; data: PageData; onNavigate: (u: string) => void }) {
  const go = (u: string) => onNavigate(u);
  return (
    <div className="bg-white text-[13px] text-slate-700">
      {/* site header */}
      <div className="border-b border-slate-200 px-8 pb-3 pt-5">
        <div className="flex items-center justify-between">
          <span className="font-serif text-[26px] font-bold lowercase tracking-tight text-slate-900">{data.wordmark}</span>
          <div className="flex items-center gap-5 text-[12.5px] text-slate-600">
            {data.nav.map((n) => (
              <span key={n} className="cursor-default">{n} <span className="text-slate-400">⌄</span></span>
            ))}
            <span className="ml-4 flex items-center gap-1.5 border-l border-slate-200 pl-4">
              <Icon name="search" size={13} /> Search
            </span>
            <span className="cursor-default">Sign in</span>
          </div>
        </div>
      </div>
      {data.redRule && <div className="h-[3px] bg-[#c42b2b]" />}

      <div className="flex gap-8 px-8 py-6">
        {/* main column */}
        <div className="min-w-0 flex-1">
          <div className="mb-4 flex items-center gap-1.5 text-[12px] text-slate-500">
            {data.crumb.map((c, i) => (
              <span key={c} className="flex items-center gap-1.5">
                {i > 0 && <span className="text-slate-300">›</span>}
                {i < data.crumb.length - 1 ? <PageLink onClick={() => go(url)}>{c}</PageLink> : <span className="text-slate-700">{c}</span>}
              </span>
            ))}
          </div>
          <div className="mb-2 flex items-center gap-2.5 text-[12px]">
            <span className="text-slate-600">{data.kicker.label}</span>
            <PageLink onClick={() => go(url)} className="border-b border-[#d97706] font-medium text-[#d97706]">{data.kicker.accent}</PageLink>
            <span className="text-slate-400">|</span>
            <span className="text-slate-500">Published: {data.kicker.extra}</span>
          </div>
          <h1 className="mb-3 font-serif text-[27px] font-bold leading-tight text-slate-900">{data.title}</h1>
          <p className="mb-2 text-[13px]">
            {data.authors.map((a, i) => (
              <span key={a}>
                {i > 0 && ", "}
                <PageLink onClick={() => go(`lumen://search?q=${encodeURIComponent(a)}`)}>{a}</PageLink>
                <sup className="text-[10px] text-slate-400">{i + 1}</sup>
              </span>
            ))}
          </p>
          <p className="mb-3 text-[12.5px]">
            <PageLink onClick={() => go(url)}>{data.citation.journal}</PageLink>
            <span className="text-slate-600">{data.citation.rest}</span>
            <span className="mx-2 text-slate-300">|</span>
            <PageLink onClick={() => go(url)}>Cite this article</PageLink>
          </p>
          <div className="mb-6 flex items-center gap-3 border-b border-slate-100 pb-4 text-[12px]">
            {data.stats.map((s) => (
              <span key={s.l} className="text-slate-500">
                <strong className="mr-1 text-[13px] text-slate-800">{s.v}</strong> {s.l}
              </span>
            ))}
            <PageLink onClick={() => go(url)} className="ml-1">Metrics</PageLink>
          </div>

          <h2 className="mb-2 font-serif text-[19px] font-bold text-slate-900">Abstract</h2>
          <p className="mb-5 font-serif text-[14.5px] leading-[1.75] text-slate-800">{data.abstract}</p>

          {data.keywords && (
            <>
              <h3 className="mb-2 text-[13px] font-bold text-slate-900">Key words</h3>
              <div className="mb-5 flex flex-wrap gap-1.5">
                {data.keywords.map((k) => (
                  <span key={k} className="rounded bg-[#eef3fc] px-2 py-0.5 text-[11.5px] text-[#1a56c4]">{k}</span>
                ))}
              </div>
            </>
          )}

          {data.references && (
            <>
              <h3 className="mb-2 text-[13px] font-bold text-slate-900">References</h3>
              <ol className="space-y-1.5">
                {data.references.map((r) => (
                  <li key={r.n} className="font-serif text-[13px] leading-relaxed text-slate-600">
                    {r.n}. {r.text}
                  </li>
                ))}
              </ol>
            </>
          )}
        </div>

        {/* aside */}
        <aside className="hidden w-52 shrink-0 space-y-4 lg:block">
          <div className="rounded-md border border-slate-200 bg-slate-50/60 px-3.5 py-3">
            <p className="mb-2 text-[12px] font-bold text-slate-800">Sections</p>
            <ul className="space-y-1.5 text-[12.5px]">
              {data.sections.map((s, i) => (
                <li key={s}>
                  <span
                    className={i === 0 ? "cursor-default rounded bg-[#e3e9f7] px-1 font-medium text-[#1a56c4]" : ""}
                  >
                    <PageLink onClick={() => go(`${url}#${s.toLowerCase()}`)}>{s}</PageLink>
                  </span>
                </li>
              ))}
            </ul>
          </div>
          {data.downloads && (
            <div className="rounded-md border border-slate-200 px-3.5 py-3">
              <p className="mb-2 text-[12px] font-bold text-slate-800">Download</p>
              <ul className="space-y-2 text-[12.5px]">
                {data.downloads.map((d) => (
                  <li key={d.label} className="flex items-center gap-2 text-[#1a56c4]">
                    <Icon name={d.icon} size={13} className="text-slate-500" /> {d.label}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </aside>
      </div>
    </div>
  );
}

function SearchResultsPage({ url, onNavigate }: { url: string; onNavigate: (u: string) => void }) {
  const q = new URL(url).searchParams.get("q") ?? "";
  const results = [
    {
      title: "Proton gradients and ATP synthase",
      href: MOCK_PAGE_URL_B,
      desc: "A review of proton translocation, chloroplast ATP synthase structure and gradient regulation during chemiosmosis.",
    },
    {
      title: "Mechanism of ATP production during chemiosmosis",
      href: MOCK_PAGE_URL_A,
      desc: "Mechanistic analysis of how the proton motive force drives ATP synthase activity, with structural and kinetic data.",
    },
    {
      title: "Chemiosmosis — Wikipedia",
      href: "https://en.wikipedia.org/wiki/Chemiosmosis",
      desc: "Chemiosmosis is the movement of ions across a semipermeable membrane, down their electrochemical gradient…",
    },
    {
      title: "Oxidative Phosphorylation and ATP Synthesis (PDF)",
      href: "https://www.nature.com/articles/srm202145",
      desc: "Nelson et al. (2021). Review of F1Fo ATP synthase: proton stoichiometry, c-ring structure and uncoupling.",
    },
  ];
  return (
    <div className="bg-white px-10 py-8">
      <h1 className="mb-1 text-[19px] font-semibold text-slate-900">
        Search results for “{q || "…"}”
      </h1>
      <p className="mb-6 text-[12px] text-slate-400">Lumen mock index — the real search pipeline is a future integration point.</p>
      <ul className="max-w-[640px] space-y-5">
        {results.map((r) => (
          <li key={r.href}>
            <PageLink onClick={() => onNavigate(r.href)} className="text-[15px] font-medium">
              {r.title}
            </PageLink>
            <p className="text-[12px] text-[#188038]">{r.href.replace("https://", "")}</p>
            <p className="mt-0.5 text-[12.5px] leading-relaxed text-slate-600">{r.desc}</p>
          </li>
        ))}
      </ul>
    </div>
  );
}

function NewTabPage({ onNavigate }: { onNavigate: (u: string) => void }) {
  const quick = [
    { label: "Paper A — Mechanism of ATP production", url: MOCK_PAGE_URL_A },
    { label: "Paper B — Proton gradients and ATP synthase", url: MOCK_PAGE_URL_B },
    { label: "Search “chemiosmosis ATP synthesis”", url: "lumen://search?q=chemiosmosis%20ATP%20synthesis" },
  ];
  return (
    <div className="flex min-h-[320px] flex-col items-center justify-center bg-white px-10 py-10">
      <p className="mb-1 font-serif text-[22px] font-semibold text-slate-900">New tab</p>
      <p className="mb-6 text-[12.5px] text-slate-500">
        Continue your research session, or jump back to a source.
      </p>
      <div className="w-full max-w-[420px] space-y-1.5">
        {quick.map((q) => (
          <button
            key={q.url}
            type="button"
            onClick={() => onNavigate(q.url)}
            className="flex w-full items-center gap-3 rounded-lg border border-slate-200 px-4 py-2.5 text-left text-[13px] text-slate-700 transition-colors hover:border-iris-300 hover:bg-iris-50/50"
          >
            <Icon name="globe" size={14} className="text-slate-400" />
            {q.label}
          </button>
        ))}
      </div>
      <p className="mt-8 max-w-[440px] rounded-lg border border-dashed border-slate-300 bg-slate-50 px-4 py-3 text-center text-[11.5px] leading-relaxed text-slate-500">
        BrowserView placeholder. Real web pages are rendered by the browser engine, which plugs in
        here — see <span className="font-medium text-slate-700">docs/architecture.md §Browser</span>.
      </p>
    </div>
  );
}

export default function MockWebPage({
  url,
  onNavigate,
}: {
  url: string;
  onNavigate: (u: string) => void;
}) {
  if (url === "lumen://newtab") return <NewTabPage onNavigate={onNavigate} />;
  if (url.startsWith("lumen://search")) return <SearchResultsPage url={url} onNavigate={onNavigate} />;
  const data = PAGES[url];
  if (data) return <ArticlePage url={url} data={data} onNavigate={onNavigate} />;
  // generic fallback for unknown URLs
  return (
    <div className="flex min-h-[320px] flex-col items-center justify-center bg-white px-10 py-10 text-center">
      <div className="mb-3 flex h-11 w-11 items-center justify-center rounded-full bg-slate-100 text-slate-400">
        <Icon name="globe" size={20} />
      </div>
      <p className="mb-1 text-[15px] font-semibold text-slate-800">{url.replace(/^https?:\/\//, "").split("/")[0]}</p>
      <p className="max-w-[380px] text-[12.5px] leading-relaxed text-slate-500">
        Web content placeholder for this address. The mock index only ships two demo pages — any other
        URL would be handed to the real browser engine (integration point in docs/architecture.md).
      </p>
      <button
        type="button"
        onClick={() => onNavigate(MOCK_PAGE_URL_A)}
        className="mt-4 text-[12.5px] text-[#1a56c4] hover:underline"
      >
        Go back to a demo page
      </button>
    </div>
  );
}
