import React, { useState, useRef, useEffect, useMemo } from "react";
import { Filter, X, HelpCircle, Sparkles } from "lucide-react";

export interface FieldDef {
  name: string;
  type: "string" | "number";
  description: string;
  examples: string[];
}

interface Suggestion {
  id: string;
  label: string;
  insertText: string;
  detail: string;
  category: "Field" | "Operator" | "Method" | "Logic" | "Preset";
}

interface SmartFilterInputProps {
  value: string;
  onApply: (query: string) => void;
  fields: FieldDef[];
  presets?: { label: string; query: string }[];
  placeholder?: string;
  error?: string | null;
}

export function SmartFilterInput({
  value: initialValue,
  onApply,
  fields,
  presets = [],
  placeholder = "Filter components (e.g. kv >= 1900 && manufacturer.contains('T-Motor'))...",
  error,
}: SmartFilterInputProps) {
  const [query, setQuery] = useState(initialValue);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [showHelp, setShowHelp] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setQuery(initialValue);
  }, [initialValue]);

  // Derive suggestions based on the current cursor position / input token
  const suggestions: Suggestion[] = useMemo(() => {
    const trimmed = query;
    if (!trimmed.trim()) {
      // Suggest top fields and presets
      return [
        ...fields.map((f) => ({
          id: `field-${f.name}`,
          label: f.name,
          insertText: f.type === "string" ? `${f.name}.contains("")` : `${f.name} >= `,
          detail: `${f.type} - ${f.description}`,
          category: "Field" as const,
        })),
      ];
    }

    const input = inputRef.current;
    const cursorPos = input ? input.selectionStart ?? trimmed.length : trimmed.length;
    const textBefore = trimmed.slice(0, cursorPos);

    // Check if user is typing after dot: fieldName.method
    const dotMatch = textBefore.match(/([a-zA-Z0-9_]+)\.([a-zA-Z]*)$/);
    if (dotMatch) {
      const fieldName = dotMatch[1];
      const methodPrefix = dotMatch[2].toLowerCase();
      const field = fields.find((f) => f.name.toLowerCase() === fieldName.toLowerCase());

      if (field && field.type === "string") {
        const methods = [
          { name: "contains", insert: `contains("")`, desc: `String contains substring` },
          { name: "startsWith", insert: `startsWith("")`, desc: `String starts with prefix` },
          { name: "endsWith", insert: `endsWith("")`, desc: `String ends with suffix` },
        ];
        return methods
          .filter((m) => m.name.toLowerCase().startsWith(methodPrefix))
          .map((m) => ({
            id: `method-${m.name}`,
            label: `.${m.name}("")`,
            insertText: `${m.name}("")`,
            detail: m.desc,
            category: "Method" as const,
          }));
      }
    }

    // Check if user just typed a field name followed by space: "kv " or "manufacturer "
    const fieldSpaceMatch = textBefore.match(/([a-zA-Z0-9_]+)\s+$/);
    if (fieldSpaceMatch) {
      const field = fields.find((f) => f.name.toLowerCase() === fieldSpaceMatch[1].toLowerCase());
      if (field) {
        if (field.type === "number") {
          return [
            { id: "op->=", label: ">= (Greater than or equal)", insertText: ">= ", detail: "Number comparison", category: "Operator" },
            { id: "op-<=", label: "<= (Less than or equal)", insertText: "<= ", detail: "Number comparison", category: "Operator" },
            { id: "op-==", label: "== (Exact equal)", insertText: "== ", detail: "Number equality", category: "Operator" },
            { id: "op->", label: "> (Greater than)", insertText: "> ", detail: "Number comparison", category: "Operator" },
            { id: "op-<", label: "< (Less than)", insertText: "< ", detail: "Number comparison", category: "Operator" },
            { id: "op-!=", label: "!= (Not equal)", insertText: "!= ", detail: "Number inequality", category: "Operator" },
          ];
        } else {
          return [
            { id: "op-==", label: '== "" (Exact match)', insertText: '== ""', detail: "Exact string match", category: "Operator" },
            { id: "op-!=", label: '!= "" (Not equal)', insertText: '!= ""', detail: "String inequality", category: "Operator" },
          ];
        }
      }
    }

    // Check if expression looks complete and user might want logical operators: " && " or " || "
    if (/["'0-9)]\s*$/.test(textBefore)) {
      return [
        { id: "logic-and", label: "&& (AND)", insertText: " && ", detail: "Combine with AND condition", category: "Logic" },
        { id: "logic-or", label: "|| (OR)", insertText: " || ", detail: "Combine with OR condition", category: "Logic" },
      ];
    }

    // Extract current token being typed
    const tokenMatch = textBefore.match(/([a-zA-Z0-9_]*)$/);
    const currentToken = tokenMatch ? tokenMatch[1].toLowerCase() : "";

    const matchedFields = fields
      .filter((f) => f.name.toLowerCase().includes(currentToken))
      .map((f) => ({
        id: `field-${f.name}`,
        label: f.name,
        insertText: f.type === "string" ? `${f.name}.contains("")` : `${f.name} >= `,
        detail: `${f.type} - ${f.description}`,
        category: "Field" as const,
      }));

    return matchedFields;
  }, [query, fields]);

  // Keep selectedIndex in bounds
  useEffect(() => {
    setSelectedIndex(0);
  }, [suggestions]);

  // Scroll selected suggestion into view
  useEffect(() => {
    if (!showSuggestions || selectedIndex < 0 || !listRef.current) return;
    const itemEl = listRef.current.children[selectedIndex] as HTMLElement | undefined;
    if (itemEl && typeof itemEl.scrollIntoView === "function") {
      itemEl.scrollIntoView({ block: "nearest" });
    }
  }, [selectedIndex, showSuggestions]);

  const handleApply = (queryToApply?: string) => {
    const q = (queryToApply !== undefined ? queryToApply : query).trim();
    setShowSuggestions(false);
    onApply(q);
  };

  const handleClear = () => {
    setQuery("");
    setShowSuggestions(false);
    onApply("");
    inputRef.current?.focus();
  };

  const insertSuggestion = (suggestion: Suggestion) => {
    const input = inputRef.current;
    if (!input) return;

    const cursorPos = input.selectionStart ?? query.length;
    const textBefore = query.slice(0, cursorPos);
    const textAfter = query.slice(cursorPos);

    let newText = "";
    let newCursor = 0;

    if (suggestion.category === "Method") {
      // Replace after the dot
      const dotIndex = textBefore.lastIndexOf(".");
      newText = textBefore.slice(0, dotIndex + 1) + suggestion.insertText + textAfter;
      newCursor = dotIndex + 1 + suggestion.insertText.length - 2; // place cursor inside quotes ("")
    } else if (suggestion.category === "Field") {
      // Replace last word token
      const lastTokenMatch = textBefore.match(/([a-zA-Z0-9_]*)$/);
      const tokenLen = lastTokenMatch ? lastTokenMatch[1].length : 0;
      const baseBefore = textBefore.slice(0, textBefore.length - tokenLen);
      newText = baseBefore + suggestion.insertText + textAfter;
      newCursor = (baseBefore + suggestion.insertText).length;
      if (suggestion.insertText.endsWith('("")')) {
        newCursor -= 2; // inside quotes
      }
    } else {
      // Append / insert operator
      newText = textBefore + suggestion.insertText + textAfter;
      newCursor = textBefore.length + suggestion.insertText.length;
      if (suggestion.insertText.endsWith('""')) {
        newCursor -= 1;
      }
    }

    setQuery(newText);
    setShowSuggestions(false);

    setTimeout(() => {
      if (inputRef.current) {
        inputRef.current.focus();
        inputRef.current.setSelectionRange(newCursor, newCursor);
      }
    }, 10);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      setShowSuggestions(false);
      handleApply();
      return;
    }

    if (showSuggestions && suggestions.length > 0) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSelectedIndex((prev) => (prev + 1) % suggestions.length);
        return;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        setSelectedIndex((prev) => (prev - 1 + suggestions.length) % suggestions.length);
        return;
      }
      if (e.key === "Tab") {
        e.preventDefault();
        insertSuggestion(suggestions[selectedIndex]);
        return;
      }
      if (e.key === "Escape") {
        setShowSuggestions(false);
        return;
      }
    }
  };

  return (
    <div className="w-full mb-6">
      {/* Main Filter Bar */}
      <div className="relative flex items-center">
        <div className="relative flex-1">
          <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-zinc-400">
            <Filter size={16} />
          </div>
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setShowSuggestions(true);
            }}
            onFocus={() => setShowSuggestions(true)}
            onBlur={() => {
              // Delay hide to allow mouse click on dropdown items
              setTimeout(() => setShowSuggestions(false), 200);
            }}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            className="w-full pl-9 pr-24 py-2.5 bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-lg text-sm text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-all font-mono"
          />

          <div className="absolute inset-y-0 right-0 pr-2 flex items-center gap-1">
            {query && (
              <button
                type="button"
                onClick={handleClear}
                title="Clear filter"
                className="p-1 rounded-md text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
              >
                <X size={15} />
              </button>
            )}
            <button
              type="button"
              onClick={() => setShowHelp(!showHelp)}
              title="CEL Syntax Guide"
              className={`p-1.5 rounded-md text-xs font-medium transition-colors flex items-center gap-1 ${
                showHelp
                  ? "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300"
                  : "text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-800"
              }`}
            >
              <HelpCircle size={15} />
            </button>
          </div>
        </div>

        <button
          type="button"
          onClick={() => handleApply()}
          className="ml-2 px-4 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors shadow-sm cursor-pointer"
        >
          Apply
        </button>
      </div>

      {/* Autocomplete Dropdown */}
      {showSuggestions && suggestions.length > 0 && (
        <div
          ref={dropdownRef}
          className="absolute z-50 mt-1.5 w-full max-w-2xl bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-lg shadow-xl overflow-hidden text-sm"
        >
          <div className="px-3 py-1.5 text-xs font-semibold uppercase tracking-wider text-zinc-400 bg-zinc-50 dark:bg-zinc-950/50 border-b border-zinc-200 dark:border-zinc-800 flex justify-between items-center">
            <span>Query Autocompletion</span>
            <span className="text-[10px] text-zinc-400 font-normal">Use ↑↓ to navigate, Tab to insert, Enter to apply</span>
          </div>
          <div ref={listRef} className="max-h-60 overflow-y-auto divide-y divide-zinc-100 dark:divide-zinc-800/50">
            {suggestions.map((item, idx) => (
              <div
                key={item.id}
                onMouseDown={(e) => {
                  e.preventDefault(); // prevent input blur
                  insertSuggestion(item);
                }}
                onMouseEnter={() => setSelectedIndex(idx)}
                className={`px-3 py-2 flex items-center justify-between cursor-pointer transition-colors ${
                  idx === selectedIndex
                    ? "bg-blue-50 dark:bg-blue-950/60 text-blue-900 dark:text-blue-100"
                    : "hover:bg-zinc-50 dark:hover:bg-zinc-800/40 text-zinc-700 dark:text-zinc-300"
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <span
                    className={`text-[10px] uppercase font-bold px-1.5 py-0.5 rounded ${
                      item.category === "Field"
                        ? "bg-emerald-100 dark:bg-emerald-950/70 text-emerald-700 dark:text-emerald-300"
                        : item.category === "Method"
                        ? "bg-purple-100 dark:bg-purple-950/70 text-purple-700 dark:text-purple-300"
                        : item.category === "Operator"
                        ? "bg-amber-100 dark:bg-amber-950/70 text-amber-700 dark:text-amber-300"
                        : "bg-blue-100 dark:bg-blue-950/70 text-blue-700 dark:text-blue-300"
                    }`}
                  >
                    {item.category}
                  </span>
                  <span className="font-mono font-medium">{item.label}</span>
                </div>
                <span className="text-xs text-zinc-400">{item.detail}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Preset Chips */}
      {presets.length > 0 && (
        <div className="flex flex-wrap items-center gap-1.5 mt-2">
          <span className="text-xs text-zinc-400 flex items-center gap-1 mr-1">
            <Sparkles size={12} />
            Quick Filters:
          </span>
          {presets.map((preset) => (
            <button
              key={preset.query}
              type="button"
              onClick={() => {
                setQuery(preset.query);
                handleApply(preset.query);
              }}
              className={`text-xs px-2.5 py-1 rounded-full border transition-colors font-mono cursor-pointer ${
                query === preset.query
                  ? "bg-blue-50 dark:bg-blue-950/60 border-blue-300 dark:border-blue-800 text-blue-700 dark:text-blue-300"
                  : "bg-zinc-50 dark:bg-zinc-900 border-zinc-200 dark:border-zinc-800 text-zinc-600 dark:text-zinc-400 hover:border-zinc-300 dark:hover:border-zinc-700"
              }`}
            >
              {preset.label}
            </button>
          ))}
        </div>
      )}

      {/* Error message */}
      {error && (
        <div className="mt-2 text-xs text-red-500 dark:text-red-400 flex items-center gap-1.5">
          <span>Invalid filter expression: {error}</span>
        </div>
      )}

      {/* Collapsible Syntax Guide */}
      {showHelp && (
        <div className="mt-3 p-4 bg-zinc-50 dark:bg-zinc-900/70 border border-zinc-200 dark:border-zinc-800 rounded-lg text-xs space-y-3">
          <div className="flex items-center justify-between border-b border-zinc-200 dark:border-zinc-800 pb-2">
            <h4 className="font-semibold text-zinc-800 dark:text-zinc-200 flex items-center gap-1.5">
              <span>Common Expression Language (CEL) Filter Syntax</span>
            </h4>
            <button
              type="button"
              onClick={() => setShowHelp(false)}
              className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200"
            >
              <X size={14} />
            </button>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <p className="font-semibold text-zinc-700 dark:text-zinc-300 mb-1">String Matching</p>
              <ul className="space-y-1 font-mono text-[11px] text-zinc-600 dark:text-zinc-400">
                <li><code className="bg-zinc-200 dark:bg-zinc-800 px-1 py-0.5 rounded">manufacturer.contains("T-Motor")</code></li>
                <li><code className="bg-zinc-200 dark:bg-zinc-800 px-1 py-0.5 rounded">name.startsWith("Velox")</code></li>
                <li><code className="bg-zinc-200 dark:bg-zinc-800 px-1 py-0.5 rounded">manufacturer == "Emax"</code></li>
              </ul>
            </div>
            <div>
              <p className="font-semibold text-zinc-700 dark:text-zinc-300 mb-1">Numeric Comparison</p>
              <ul className="space-y-1 font-mono text-[11px] text-zinc-600 dark:text-zinc-400">
                <li><code className="bg-zinc-200 dark:bg-zinc-800 px-1 py-0.5 rounded">kv &gt;= 1900</code></li>
                <li><code className="bg-zinc-200 dark:bg-zinc-800 px-1 py-0.5 rounded">weight_g &lt; 35.0</code></li>
                <li><code className="bg-zinc-200 dark:bg-zinc-800 px-1 py-0.5 rounded">stator_diameter_mm == 22.0</code></li>
              </ul>
            </div>
            <div>
              <p className="font-semibold text-zinc-700 dark:text-zinc-300 mb-1">Compound Conditions</p>
              <ul className="space-y-1 font-mono text-[11px] text-zinc-600 dark:text-zinc-400">
                <li><code className="bg-zinc-200 dark:bg-zinc-800 px-1 py-0.5 rounded">kv &gt;= 1750 &amp;&amp; kv &lt;= 2000</code></li>
                <li><code className="bg-zinc-200 dark:bg-zinc-800 px-1 py-0.5 rounded">manufacturer == "T-Motor" || kv &gt; 2400</code></li>
              </ul>
            </div>
          </div>

          <div className="pt-2 border-t border-zinc-200 dark:border-zinc-800">
            <p className="font-semibold text-zinc-700 dark:text-zinc-300 mb-1">Available Fields for this Collection:</p>
            <div className="flex flex-wrap gap-2">
              {fields.map((f) => (
                <button
                  key={f.name}
                  type="button"
                  onClick={() => {
                    const sample = f.examples[0] || `${f.name} == `;
                    setQuery(sample);
                    handleApply(sample);
                  }}
                  className="font-mono text-[11px] bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 px-2 py-0.5 rounded hover:border-blue-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
                  title={`Click to test: ${f.examples[0] || f.name}`}
                >
                  {f.name} <span className="text-zinc-400 text-[10px]">({f.type})</span>
                </button>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
