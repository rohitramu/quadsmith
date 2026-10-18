import { Component, type ErrorInfo, type ReactNode } from "react";
import { AlertTriangle, RotateCcw } from "lucide-react";

export interface ErrorBoundaryFallbackProps {
  error: Error;
  resetErrorBoundary: () => void;
}

export interface ErrorBoundaryProps {
  children: ReactNode;
  /**
   * Presentation variant:
   * - "page": Centered route/page error card with Try Again & Reload Page buttons.
   * - "panel": Compact embedded alert card for isolated widgets/panels (e.g. Build Evaluator, Forge Steps).
   */
  variant?: "page" | "panel";
  title?: string;
  description?: string;
  resetLabel?: string;
  /**
   * When any value in resetKeys changes while the boundary is in an error state,
   * the boundary automatically resets and re-renders children.
   */
  resetKeys?: ReadonlyArray<unknown>;
  onReset?: () => void;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  fallback?: ReactNode | ((props: ErrorBoundaryFallbackProps) => ReactNode);
}

interface ErrorBoundaryState {
  error: Error | null;
}

function areResetKeysEqual(
  prev: ReadonlyArray<unknown> = [],
  next: ReadonlyArray<unknown> = [],
): boolean {
  if (prev.length !== next.length) return false;
  for (let i = 0; i < prev.length; i++) {
    if (!Object.is(prev[i], next[i])) {
      return false;
    }
  }
  return true;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  override state: ErrorBoundaryState = {
    error: null,
  };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  override componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    this.props.onError?.(error, errorInfo);
  }

  override componentDidUpdate(prevProps: ErrorBoundaryProps): void {
    if (
      this.state.error !== null &&
      this.props.resetKeys &&
      prevProps.resetKeys &&
      !areResetKeysEqual(prevProps.resetKeys, this.props.resetKeys)
    ) {
      this.setState({ error: null });
    }
  }

  resetErrorBoundary = (): void => {
    this.props.onReset?.();
    this.setState({ error: null });
  };

  override render(): ReactNode {
    const { error } = this.state;
    const {
      children,
      fallback,
      variant = "page",
      title,
      description,
      resetLabel = "Try Again",
    } = this.props;

    if (error) {
      if (typeof fallback === "function") {
        return fallback({ error, resetErrorBoundary: this.resetErrorBoundary });
      }
      if (fallback !== undefined) {
        return fallback;
      }

      if (variant === "panel") {
        return (
          <div
            role="alert"
            data-testid="error-boundary-fallback"
            className="rounded-2xl border border-red-200 dark:border-red-900/60 bg-red-50/70 dark:bg-red-950/30 p-4 space-y-3 shadow-xs"
          >
            <div className="flex items-start gap-2.5">
              <AlertTriangle className="text-red-600 dark:text-red-400 shrink-0 mt-0.5" size={18} />
              <div className="min-w-0 flex-1">
                <h3 className="text-xs font-bold uppercase tracking-wider text-red-900 dark:text-red-200">
                  {title || "Component Error"}
                </h3>
                <p className="text-xs text-red-700 dark:text-red-300/90 mt-1 leading-relaxed">
                  {description ||
                    "An unexpected error occurred in this panel. Your other selections and page state remain intact."}
                </p>
                {error.message && (
                  <div className="mt-2 px-2.5 py-1.5 rounded-lg bg-red-100/80 dark:bg-red-950/70 border border-red-200/80 dark:border-red-900/50 text-[11px] font-mono text-red-800 dark:text-red-300 break-words">
                    {error.message}
                  </div>
                )}
              </div>
            </div>
            <div className="flex justify-end">
              <button
                type="button"
                onClick={this.resetErrorBoundary}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-red-600 hover:bg-red-700 text-white text-xs font-semibold transition-colors cursor-pointer shadow-2xs"
              >
                <RotateCcw size={13} />
                <span>{resetLabel}</span>
              </button>
            </div>
          </div>
        );
      }

      return (
        <div
          role="alert"
          data-testid="error-boundary-fallback"
          className="max-w-xl mx-auto my-8 rounded-2xl border border-red-200 dark:border-red-900/60 bg-white dark:bg-zinc-900 p-6 shadow-sm space-y-4"
        >
          <div className="flex items-start gap-3.5">
            <div className="w-10 h-10 rounded-xl bg-red-500/10 border border-red-500/20 flex items-center justify-center shrink-0">
              <AlertTriangle className="text-red-600 dark:text-red-400" size={20} />
            </div>
            <div className="min-w-0 flex-1">
              <h2 className="text-base font-bold text-zinc-900 dark:text-zinc-100">
                {title || "Something went wrong"}
              </h2>
              <p className="text-xs sm:text-sm text-zinc-600 dark:text-zinc-400 mt-1 leading-relaxed">
                {description ||
                  "An unexpected error occurred while rendering this section. You can try again or navigate to another page."}
              </p>
              {error.message && (
                <div className="mt-3 px-3 py-2 rounded-xl bg-zinc-100 dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 text-xs font-mono text-red-600 dark:text-red-400 break-words">
                  {error.message}
                </div>
              )}
            </div>
          </div>
          <div className="flex items-center justify-end gap-2 pt-2 border-t border-zinc-200 dark:border-zinc-800">
            <button
              type="button"
              onClick={() => window.location.reload()}
              className="px-3.5 py-2 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-800/60 hover:bg-zinc-100 dark:hover:bg-zinc-800 text-zinc-700 dark:text-zinc-300 text-xs font-semibold transition-colors cursor-pointer"
            >
              Reload Page
            </button>
            <button
              type="button"
              onClick={this.resetErrorBoundary}
              className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-xs font-semibold transition-colors cursor-pointer shadow-xs"
            >
              <RotateCcw size={13} />
              <span>{resetLabel}</span>
            </button>
          </div>
        </div>
      );
    }

    return children;
  }
}
