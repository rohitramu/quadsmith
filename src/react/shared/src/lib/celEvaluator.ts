import type { FieldDef } from "../components/SmartFilterInput";

export type CelRecord = Record<string, string | number | boolean | bigint | null | undefined>;

interface Token {
  type: "ident" | "string" | "number" | "boolean" | "op" | "lparen" | "rparen" | "dot" | "eof";
  value: string;
}

function tokenizeCel(input: string): Token[] {
  const tokens: Token[] = [];
  let i = 0;

  while (i < input.length) {
    const ch = input[i];
    if (/\s/.test(ch)) {
      i++;
      continue;
    }

    if (ch === "(") {
      tokens.push({ type: "lparen", value: "(" });
      i++;
      continue;
    }
    if (ch === ")") {
      tokens.push({ type: "rparen", value: ")" });
      i++;
      continue;
    }
    if (ch === ".") {
      tokens.push({ type: "dot", value: "." });
      i++;
      continue;
    }

    const twoChar = input.slice(i, i + 2);
    if (["==", "!=", "<=", ">=", "&&", "||"].includes(twoChar)) {
      tokens.push({ type: "op", value: twoChar });
      i += 2;
      continue;
    }

    if (["<", ">", "!"].includes(ch)) {
      tokens.push({ type: "op", value: ch });
      i++;
      continue;
    }

    if (ch === '"' || ch === "'") {
      const quote = ch;
      i++;
      let str = "";
      while (i < input.length && input[i] !== quote) {
        if (input[i] === "\\" && i + 1 < input.length) {
          i++;
        }
        str += input[i];
        i++;
      }
      if (i >= input.length || input[i] !== quote) {
        throw new Error("Unterminated string literal");
      }
      i++;
      tokens.push({ type: "string", value: str });
      continue;
    }

    if (/[0-9]/.test(ch) || (ch === "-" && /[0-9]/.test(input[i + 1] || ""))) {
      let numStr = ch;
      i++;
      while (i < input.length && /[0-9.]/.test(input[i])) {
        numStr += input[i];
        i++;
      }
      if (Number.isNaN(Number(numStr))) {
        throw new Error(`Invalid numeric literal: ${numStr}`);
      }
      tokens.push({ type: "number", value: numStr });
      continue;
    }

    if (/[a-zA-Z_]/.test(ch)) {
      let ident = ch;
      i++;
      while (i < input.length && /[a-zA-Z0-9_]/.test(input[i])) {
        ident += input[i];
        i++;
      }
      if (ident === "true" || ident === "false") {
        tokens.push({ type: "boolean", value: ident });
      } else {
        tokens.push({ type: "ident", value: ident });
      }
      continue;
    }

    throw new Error(`Unexpected character '${ch}' in CEL expression`);
  }

  tokens.push({ type: "eof", value: "" });
  return tokens;
}

type AstNode =
  | { kind: "literal"; value: string | number | boolean }
  | { kind: "ident"; name: string }
  | { kind: "unary"; op: "!"; arg: AstNode }
  | { kind: "binary"; op: string; left: AstNode; right: AstNode }
  | {
      kind: "method";
      target: string;
      method: "contains" | "startsWith" | "endsWith";
      arg: string;
    };

function parseCel(expr: string, allowedFields?: FieldDef[]): AstNode {
  const tokens = tokenizeCel(expr);
  let pos = 0;

  const peek = () => tokens[pos];
  const consume = () => tokens[pos++];

  const validFieldMap = allowedFields ? new Map(allowedFields.map((f) => [f.name, f])) : null;

  function validateField(name: string) {
    if (validFieldMap && !validFieldMap.has(name)) {
      throw new Error(`Unknown field '${name}'`);
    }
  }

  function parseOr(): AstNode {
    let left = parseAnd();
    while (peek().type === "op" && peek().value === "||") {
      consume();
      const right = parseAnd();
      left = { kind: "binary", op: "||", left, right };
    }
    return left;
  }

  function parseAnd(): AstNode {
    let left = parseUnary();
    while (peek().type === "op" && peek().value === "&&") {
      consume();
      const right = parseUnary();
      left = { kind: "binary", op: "&&", left, right };
    }
    return left;
  }

  function parseUnary(): AstNode {
    if (peek().type === "op" && peek().value === "!") {
      consume();
      return { kind: "unary", op: "!", arg: parseUnary() };
    }
    return parseComparison();
  }

  function parseComparison(): AstNode {
    const left = parsePrimary();
    if (peek().type === "op" && ["==", "!=", "<", "<=", ">", ">="].includes(peek().value)) {
      const op = consume().value;
      const right = parsePrimary();
      return { kind: "binary", op, left, right };
    }
    return left;
  }

  function parsePrimary(): AstNode {
    const tok = peek();
    if (tok.type === "lparen") {
      consume();
      const inner = parseOr();
      if (peek().type !== "rparen") {
        throw new Error("Expected closing ')'");
      }
      consume();
      return inner;
    }

    if (tok.type === "string") {
      consume();
      return { kind: "literal", value: tok.value };
    }
    if (tok.type === "number") {
      consume();
      return { kind: "literal", value: Number(tok.value) };
    }
    if (tok.type === "boolean") {
      consume();
      return { kind: "literal", value: tok.value === "true" };
    }
    if (tok.type === "ident") {
      const identName = consume().value;
      validateField(identName);
      if (peek().type === "dot") {
        consume(); // '.'
        const methodTok = consume();
        if (
          methodTok.type !== "ident" ||
          !["contains", "startsWith", "endsWith"].includes(methodTok.value)
        ) {
          throw new Error(
            `Unsupported string method '${methodTok.value}' (use contains, startsWith, or endsWith)`,
          );
        }
        if (consume().type !== "lparen") {
          throw new Error(`Expected '(' after .${methodTok.value}`);
        }
        const argTok = consume();
        if (argTok.type !== "string") {
          throw new Error(`Method .${methodTok.value}() requires a string argument`);
        }
        if (consume().type !== "rparen") {
          throw new Error(`Expected ')' after .${methodTok.value}("${argTok.value}")`);
        }
        return {
          kind: "method",
          target: identName,
          method: methodTok.value as "contains" | "startsWith" | "endsWith",
          arg: argTok.value,
        };
      }
      return { kind: "ident", name: identName };
    }

    throw new Error(`Unexpected token '${tok.value || tok.type}'`);
  }

  const ast = parseOr();
  if (peek().type !== "eof") {
    throw new Error(`Unexpected trailing token '${peek().value}'`);
  }
  return ast;
}

function evalAst(node: AstNode, rec: CelRecord): string | number | boolean {
  switch (node.kind) {
    case "literal":
      return node.value;
    case "ident": {
      const raw = rec[node.name];
      if (typeof raw === "bigint") return Number(raw);
      if (raw === null || raw === undefined) return "";
      return raw;
    }
    case "unary":
      return !Boolean(evalAst(node.arg, rec));
    case "method": {
      const targetVal = String(rec[node.target] ?? "");
      if (node.method === "contains") return targetVal.includes(node.arg);
      if (node.method === "startsWith") return targetVal.startsWith(node.arg);
      return targetVal.endsWith(node.arg);
    }
    case "binary": {
      if (node.op === "&&") {
        return Boolean(evalAst(node.left, rec)) && Boolean(evalAst(node.right, rec));
      }
      if (node.op === "||") {
        return Boolean(evalAst(node.left, rec)) || Boolean(evalAst(node.right, rec));
      }
      const l = evalAst(node.left, rec);
      const r = evalAst(node.right, rec);
      switch (node.op) {
        case "==":
          return l === r;
        case "!=":
          return l !== r;
        case "<":
          return Number(l) < Number(r);
        case "<=":
          return Number(l) <= Number(r);
        case ">":
          return Number(l) > Number(r);
        case ">=":
          return Number(l) >= Number(r);
        default:
          return false;
      }
    }
  }
}

export function filterAndSortRecords<T>(
  items: T[],
  toRecord: (item: T) => CelRecord,
  filterExpr: string,
  sortSpec: string,
  allowedFields?: FieldDef[],
): { items: T[]; error: string | null } {
  let filtered = [...items];
  const trimmed = filterExpr.trim();

  if (trimmed) {
    try {
      const ast = parseCel(trimmed, allowedFields);
      filtered = filtered.filter((item) => Boolean(evalAst(ast, toRecord(item))));
    } catch (err) {
      return {
        items,
        error: err instanceof Error ? err.message : "Invalid CEL filter expression",
      };
    }
  }

  if (sortSpec.trim()) {
    const desc = sortSpec.startsWith("^") || sortSpec.startsWith("-");
    const field = desc ? sortSpec.slice(1) : sortSpec;
    filtered.sort((a, b) => {
      const recA = toRecord(a);
      const recB = toRecord(b);
      const valA = recA[field];
      const valB = recB[field];

      const normA = typeof valA === "bigint" ? Number(valA) : (valA ?? "");
      const normB = typeof valB === "bigint" ? Number(valB) : (valB ?? "");

      let cmp = 0;
      if (typeof normA === "number" && typeof normB === "number") {
        cmp = normA - normB;
      } else {
        cmp = String(normA).localeCompare(String(normB));
      }
      return desc ? -cmp : cmp;
    });
  }

  return { items: filtered, error: null };
}
