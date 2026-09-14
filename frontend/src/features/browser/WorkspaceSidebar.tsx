/**
 * Browser left sidebar — workspace navigation + research sessions.
 * "Research" is fully wired (session switching); the remaining items
 * are honest placeholders with clear future ownership (Go backend).
 */

import { useState } from "react";
import { useApp } from "../../state/appState";
import { Icon, type IconName } from "../../components/icons";
import { EmptyState, Button } from "../../components/ui";
import { cn } from "../../utils/cn";
import { truncate } from "../../utils/helpers";

const NAV: { id: string; label: string; icon: IconName }[] = [
  { id: "home", label: "Home", icon: "home" },
  { id: "research", label: "Research", icon: "flask" },
  { id: "library", label: "Library", icon: "book" },
  { id: "notes", label: "Notes", icon: "note" },
  { id: "collections", label: "Collections", icon: "layers" },
];

export default function WorkspaceSidebar() {
  const { state, dispatch } = useApp();
  const [nav, setNav] = useState("research");

  const sessions = state.browser.sessions;
  const activeSessionId = state.browser.activeSessionId;
  const pending = Object.values(state.changes).filter((c) => c.status === "pending").length;

  const highlights = Object.entries(state.documents).flatMap(([docId, doc]) =>
    doc.highlights.map((h) => ({ h, docId, title: doc.metadata.title })),
  );

  return (
    <div className="flex h-full w-60 shrink-0 flex-col border-r border-line bg-paper">
      {/* primary nav */}
      <nav className="space-y-0.5 px-2.5 pt-3">
        {NAV.map((item) => (
          <button
            key={item.id}
            type="button"
            onClick={() => setNav(item.id)}
            className={cn(
              "flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-[13px] transition-colors",
              nav === item.id
                ? "bg-iris-100/80 font-medium text-iris-700"
                : "text-ink-500 hover:bg-cream-100 hover:text-ink-900",
            )}
          >
            <Icon name={item.icon} size={15} className={nav === item.id ? "text-iris-600" : "opacity-60"} />
            {item.label}
          </button>
        ))}
      </nav>

      <div className="min-h-0 flex-1 overflow-y-auto px-2.5 py-3">
        {nav === "home" && (
          <div className="space-y-4 fade-in">
            <section>
              <h4 className="mb-1.5 px-3 text-[11px] font-semibold uppercase tracking-wide text-mute">
                Research sessions
              </h4>
              <ul className="space-y-0.5">
                {sessions.map((s) => (
                  <li key={s.id}>
                    <button
                      type="button"
                      onClick={() => {
                        dispatch({ type: "browser.session.select", id: s.id });
                        setNav("research");
                      }}
                      className="w-full rounded-md px-3 py-1.5 text-left text-[12.5px] text-ink-700 hover:bg-cream-100"
                    >
                      <span className="flex items-center gap-2">
                        <Icon name="flask" size={13} className="text-mute" />
                        {truncate(s.title, 30)}
                        <span className="ml-auto text-[11px] text-mute">{s.tabIds.length} tab{s.tabIds.length === 1 ? "" : "s"}</span>
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            </section>
            <section className="rounded-lg border border-line-soft bg-cream-50 px-3 py-2.5 text-[12px] text-ink-500">
              <div className="flex justify-between py-0.5">
                <span>Open documents</span>
                <span className="font-medium text-ink-900">{state.reader.session.tabIds.length}</span>
              </div>
              <div className="flex justify-between py-0.5">
                <span>Pending AI changes</span>
                <span className={cn("font-medium", pending ? "text-hay-600" : "text-ink-900")}>{pending}</span>
              </div>
            </section>
            <Button variant="soft" icon="bookOpen" onClick={() => dispatch({ type: "mode.set", mode: "reader" })}>
              Resume in Reader
            </Button>
          </div>
        )}

        {nav === "research" && (
          <div className="fade-in">
            <h4 className="mb-1.5 flex items-center gap-1.5 px-3 text-[11px] font-semibold uppercase tracking-wide text-mute">
              <Icon name="clock" size={12} /> Recent sessions
            </h4>
            <ul className="space-y-0.5">
              {sessions.map((s) => (
                <li key={s.id}>
                  <button
                    type="button"
                    onClick={() => dispatch({ type: "browser.session.select", id: s.id })}
                    className={cn(
                      "flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-[12.5px] transition-colors",
                      s.id === activeSessionId
                        ? "bg-iris-100/80 font-medium text-iris-700"
                        : "text-ink-700 hover:bg-cream-100",
                    )}
                  >
                    <Icon name="flask" size={13} className={s.id === activeSessionId ? "text-iris-600" : "text-mute"} />
                    <span className="truncate">{s.title}</span>
                    {s.id === activeSessionId && <span className="ml-auto h-1.5 w-1.5 shrink-0 rounded-full bg-iris-500" />}
                  </button>
                </li>
              ))}
            </ul>
          </div>
        )}

        {nav === "library" && (
          <EmptyState
            icon="book"
            title="Local document library"
            hint="Listed and indexed by the Go backend (backend/services/workspace.go). Browse local files today via Reader → Files."
          />
        )}

        {nav === "notes" && (
          <div className="fade-in">
            <h4 className="mb-1.5 px-3 text-[11px] font-semibold uppercase tracking-wide text-mute">Saved notes</h4>
            {highlights.length === 0 ? (
              <EmptyState icon="note" title="No notes yet" hint="Select text in a source document and choose “Save note”." />
            ) : (
              <ul className="space-y-2 px-3">
                {highlights.map(({ h, title }) => (
                  <li key={h.id} className="rounded-md border border-line-soft bg-cream-50 px-3 py-2">
                    <p className="text-[11px] font-medium text-iris-700">{title}</p>
                    <p className="mt-0.5 text-[12px] leading-snug text-ink-500">{truncate(h.text, 110)}</p>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}

        {nav === "collections" && (
          <EmptyState
            icon="layers"
            title="Collections"
            hint="Group sources and documents into curated collections. Storage: SQLite (backend/storage)."
          />
        )}
      </div>

      <div className="border-t border-line-soft px-2.5 py-2">
        <button
          type="button"
          onClick={() => dispatch({ type: "settings.set", open: true })}
          className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-[13px] text-ink-500 transition-colors hover:bg-cream-100 hover:text-ink-900"
        >
          <Icon name="settings" size={15} className="opacity-60" /> Settings
        </button>
      </div>
    </div>
  );
}
