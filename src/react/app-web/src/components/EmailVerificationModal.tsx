import { X } from "lucide-react";
import { EmailVerificationForm } from "./EmailVerificationForm";
import type { EmailVerificationFormProps } from "./EmailVerificationForm";

export interface EmailVerificationModalProps extends EmailVerificationFormProps {
  isOpen: boolean;
  onClose: () => void;
  maxWidthClass?: string;
}

export function EmailVerificationModal({
  isOpen,
  onClose,
  maxWidthClass = "max-w-md",
  ...formProps
}: EmailVerificationModalProps) {
  if (!isOpen) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-xs"
    >
      <div
        className={`w-full ${maxWidthClass} rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-xl p-8 space-y-6 relative`}
      >
        <button
          type="button"
          onClick={onClose}
          className="absolute top-4 right-4 p-1.5 rounded-lg text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 transition cursor-pointer"
          aria-label="Close"
        >
          <X size={18} />
        </button>

        <EmailVerificationForm {...formProps} onCancel={onClose} />
      </div>
    </div>
  );
}
