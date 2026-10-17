import React, { useState, useEffect, useRef, useMemo } from "react";
import { useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { getApiBaseUrl, getStoredAccessToken } from "@quadsmith/shared";
import { X, Upload, Plus, Trash2, AlertCircle, Loader2 } from "lucide-react";
import { getComponentCollection } from "../lib/hardwareCollections";
import { getComponentServiceDef } from "../lib/componentServices";

export interface ComponentFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  collectionId: string;
  initialData?: any;
  isClone?: boolean;
  onSuccess?: (item: any) => void;
}

function snakeToCamel(str: string): string {
  return str.replace(/_([a-z0-9])/g, (_, letter) => letter.toUpperCase());
}

function formatFieldLabel(name: string): string {
  const parts = name.split("_");
  return parts
    .map((p) => {
      if (p === "mm") return "(mm)";
      if (p === "g") return "(g)";
      if (p === "v") return "(V)";
      if (p === "a") return "(A)";
      if (p === "mw") return "(mW)";
      if (p === "mah") return "(mAh)";
      if (p === "kv") return "KV";
      if (p === "fc") return "FC";
      if (p === "esc") return "ESC";
      if (p === "vtx") return "VTX";
      if (p === "rx") return "RX";
      if (p === "tx") return "TX";
      if (p === "gps") return "GPS";
      if (p === "mcu") return "MCU";
      if (p === "imu") return "IMU";
      if (p === "dia") return "Dia.";
      if (p === "ht") return "Ht.";
      if (p === "min") return "Min";
      if (p === "max") return "Max";
      return p.charAt(0).toUpperCase() + p.slice(1);
    })
    .join(" ");
}

export function ComponentFormModal({
  isOpen,
  onClose,
  collectionId,
  initialData,
  isClone,
  onSuccess,
}: ComponentFormModalProps) {
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const colDef = useMemo(() => getComponentCollection(collectionId), [collectionId]);
  const srvDef = useMemo(() => getComponentServiceDef(collectionId), [collectionId]);

  const isEditing = Boolean((initialData?.id || initialData?.uuid) && !isClone);

  // Common metadata fields
  const [manufacturer, setManufacturer] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [primaryDisplayImage, setPrimaryDisplayImage] = useState("");
  const [referenceLinks, setReferenceLinks] = useState<{ label: string; url: string }[]>([]);

  // Subtype spec fields
  const [specValues, setSpecValues] = useState<Record<string, any>>({});

  // UI state
  const [isUploading, setIsUploading] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Filter fields to exclude common metadata
  const specFields = useMemo(() => {
    if (!colDef?.fields) return [];
    const excluded = new Set([
      "name",
      "manufacturer",
      "id",
      "uuid",
      "description",
      "primary_display_image",
      "primaryDisplayImage",
      "reference_links",
      "referenceLinks",
      "media",
      "user_uuid",
      "userUuid",
      "author_username",
      "authorUsername",
      "is_archived",
      "isArchived",
      "created_at",
      "createdAt",
      "updated_at",
      "updatedAt",
      "is_internal_only",
    ]);
    return colDef.fields.filter((f) => !excluded.has(f.name));
  }, [colDef]);

  // Populate initial values
  useEffect(() => {
    if (!isOpen) return;

    if (initialData) {
      setManufacturer(initialData.manufacturer || "");
      setName(initialData.name || "");
      setDescription(initialData.description || "");
      setPrimaryDisplayImage(initialData.primaryDisplayImage || "");
      setReferenceLinks(
        Array.isArray(initialData.referenceLinks)
          ? initialData.referenceLinks.map((l: any) => ({
              label: l.label || "",
              url: l.url || "",
            }))
          : [],
      );

      const specs: Record<string, any> = {};
      specFields.forEach((field) => {
        const camel = snakeToCamel(field.name);
        const val = initialData[camel] !== undefined ? initialData[camel] : initialData[field.name];
        if (val !== undefined && val !== null) {
          specs[camel] = val;
        }
      });
      setSpecValues(specs);
    } else {
      setManufacturer("");
      setName("");
      setDescription("");
      setPrimaryDisplayImage("");
      setReferenceLinks([]);
      setSpecValues({});
    }
    setFormError(null);
  }, [isOpen, initialData, specFields]);

  // Close on Escape key
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape" && isOpen) {
        onClose();
      }
    }
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onClose]);

  const { mutateAsync: mutateCreate } = useMutation(srvDef?.createMethod);
  const { mutateAsync: mutateUpdate } = useMutation(srvDef?.updateMethod);

  if (!isOpen) return null;

  const singularLabel = srvDef?.singularLabel || colDef?.name || "Component";

  const handleSpecChange = (fieldName: string, val: any) => {
    setSpecValues((prev) => ({
      ...prev,
      [fieldName]: val,
    }));
  };

  const handleAddLink = () => {
    setReferenceLinks((prev) => [...prev, { label: "", url: "" }]);
  };

  const handleRemoveLink = (index: number) => {
    setReferenceLinks((prev) => prev.filter((_, i) => i !== index));
  };

  const handleLinkChange = (index: number, key: "label" | "url", val: string) => {
    setReferenceLinks((prev) =>
      prev.map((item, i) => (i === index ? { ...item, [key]: val } : item)),
    );
  };

  const handleFileUpload = async (file: File) => {
    if (!file) return;
    setIsUploading(true);
    setFormError(null);
    try {
      const headers: Record<string, string> = {
        "Content-Type": file.type || "application/octet-stream",
        "X-Filename": file.name,
      };
      const token = getStoredAccessToken();
      if (token) {
        headers["Authorization"] = `Bearer ${token}`;
      }
      const resp = await fetch(`${getApiBaseUrl()}/api/blobs`, {
        method: "POST",
        headers,
        credentials: "include",
        body: file,
      });
      if (!resp.ok) {
        const errText = await resp.text();
        throw new Error(errText || "Blob upload failed");
      }
      const data = await resp.json();
      const blobUrl = data.url || (data.id ? `/blobs/${data.id}` : "");
      if (blobUrl) {
        setPrimaryDisplayImage(blobUrl);
      }
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : "Failed to upload file");
    } finally {
      setIsUploading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!srvDef) {
      setFormError(`Service not found for collection: ${collectionId}`);
      return;
    }

    if (!manufacturer.trim()) {
      setFormError("Manufacturer is required.");
      return;
    }
    if (!name.trim()) {
      setFormError("Component name is required.");
      return;
    }

    setIsSubmitting(true);
    setFormError(null);

    try {
      const payload: Record<string, any> = {
        manufacturer: manufacturer.trim(),
        name: name.trim(),
        description: description.trim(),
        primaryDisplayImage: primaryDisplayImage.trim() || undefined,
        referenceLinks: referenceLinks
          .filter((l) => l.url.trim().length > 0)
          .map((l) => ({
            label: l.label.trim(),
            url: l.url.trim(),
          })),
        ...specValues,
      };

      if (isEditing && initialData?.id) {
        payload.id = initialData.id;
      }
      if (isEditing && initialData?.uuid) {
        payload.uuid = initialData.uuid;
      }

      // Convert number specs
      specFields.forEach((field) => {
        const camel = snakeToCamel(field.name);
        if (field.type === "number" && payload[camel] !== undefined && payload[camel] !== "") {
          const num = Number(payload[camel]);
          if (!isNaN(num)) {
            payload[camel] = num;
          }
        }
      });

      let res: any;
      if (isEditing) {
        res = await mutateUpdate({
          [srvDef.paramKey]: payload,
        });
      } else {
        res = await mutateCreate({
          [srvDef.paramKey]: payload,
        });
      }

      await queryClient.invalidateQueries();
      if (onSuccess) {
        onSuccess(res);
      }
      onClose();
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : "Operation failed");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="component-form-modal-title"
      className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-in fade-in duration-150"
    >
      <div className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-2xl w-full max-w-2xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
        {/* Modal Header */}
        <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-800 flex items-center justify-between bg-zinc-50 dark:bg-zinc-900/50">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-red-600 text-white flex items-center justify-center font-bold text-sm shadow-xs">
              {isEditing ? "✎" : "+"}
            </div>
            <div>
              <h2 id="component-form-modal-title" className="text-lg font-bold tracking-tight">
                {isClone
                  ? `Create Similar ${singularLabel}`
                  : isEditing
                    ? `Edit ${singularLabel}`
                    : `Add New ${singularLabel}`}
              </h2>
              <p className="text-xs text-zinc-500">
                {isClone
                  ? `Create a new ${singularLabel.toLowerCase()} pre-filled with technical specifications.`
                  : "Enter technical specifications to catalog this component."}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close modal"
            className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 p-1.5 rounded-lg hover:bg-zinc-200 dark:hover:bg-zinc-800 transition-colors cursor-pointer"
          >
            <X size={18} />
          </button>
        </div>

        {/* Modal Body */}
        <form onSubmit={handleSubmit} className="flex-1 flex flex-col overflow-hidden">
          <div className="p-6 overflow-y-auto space-y-6 flex-1 text-sm">
            {formError && (
              <div className="p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 text-xs flex items-start gap-2">
                <AlertCircle size={16} className="shrink-0 mt-0.5" />
                <span className="leading-relaxed">{formError}</span>
              </div>
            )}

            {/* Basic Information */}
            <div className="space-y-4">
              <h3 className="font-semibold text-xs text-zinc-400 uppercase tracking-wider">
                Basic Information
              </h3>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label
                    htmlFor="input-manufacturer"
                    className="block text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1"
                  >
                    Manufacturer *
                  </label>
                  <input
                    id="input-manufacturer"
                    type="text"
                    required
                    value={manufacturer}
                    onChange={(e) => setManufacturer(e.target.value)}
                    placeholder="e.g. T-Motor, SpeedyBee, Tattu"
                    className="w-full px-3 py-2 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 focus:ring-2 focus:ring-red-500 focus:outline-none text-xs"
                  />
                </div>
                <div>
                  <label
                    htmlFor="input-model-name"
                    className="block text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1"
                  >
                    Model Name *
                  </label>
                  <input
                    id="input-model-name"
                    type="text"
                    required
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="e.g. F80 Pro, F405 V4, R-Line V5"
                    className="w-full px-3 py-2 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 focus:ring-2 focus:ring-red-500 focus:outline-none text-xs"
                  />
                </div>
              </div>

              <div>
                <label
                  htmlFor="input-description"
                  className="block text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1"
                >
                  Description
                </label>
                <textarea
                  id="input-description"
                  rows={2}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Optional overview or technical notes..."
                  className="w-full px-3 py-2 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 focus:ring-2 focus:ring-red-500 focus:outline-none text-xs"
                />
              </div>
            </div>

            {/* Technical Specifications */}
            {specFields.length > 0 && (
              <div className="space-y-4 pt-4 border-t border-zinc-200 dark:border-zinc-800">
                <h3 className="font-semibold text-xs text-zinc-400 uppercase tracking-wider">
                  {singularLabel} Specifications
                </h3>
                <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
                  {specFields.map((field) => {
                    const camel = snakeToCamel(field.name);
                    const label = formatFieldLabel(field.name);
                    const val = specValues[camel] ?? "";

                    if (field.type === "boolean") {
                      return (
                        <div key={field.name} className="flex items-center gap-2 pt-5">
                          <input
                            type="checkbox"
                            id={`field-${field.name}`}
                            checked={Boolean(val)}
                            onChange={(e) => handleSpecChange(camel, e.target.checked)}
                            className="rounded border-zinc-300 text-red-600 focus:ring-red-500 h-4 w-4"
                          />
                          <label
                            htmlFor={`field-${field.name}`}
                            className="text-xs font-medium text-zinc-700 dark:text-zinc-300 cursor-pointer"
                          >
                            {label}
                          </label>
                        </div>
                      );
                    }

                    return (
                      <div key={field.name}>
                        <label
                          className="block text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1 truncate"
                          title={field.description || label}
                        >
                          {label}
                        </label>
                        <input
                          type={field.type === "number" ? "number" : "text"}
                          step={field.type === "number" ? "any" : undefined}
                          value={val}
                          onChange={(e) => handleSpecChange(camel, e.target.value)}
                          placeholder={field.description || ""}
                          className="w-full px-3 py-2 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 focus:ring-2 focus:ring-red-500 focus:outline-none text-xs"
                        />
                      </div>
                    );
                  })}
                </div>
              </div>
            )}

            {/* Imagery & Media */}
            <div className="space-y-4 pt-4 border-t border-zinc-200 dark:border-zinc-800">
              <h3 className="font-semibold text-xs text-zinc-400 uppercase tracking-wider">
                Imagery & Media
              </h3>
              <div>
                <label className="block text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1">
                  Primary Display Image URL
                </label>
                <input
                  type="text"
                  value={primaryDisplayImage}
                  onChange={(e) => setPrimaryDisplayImage(e.target.value)}
                  placeholder="https://... or /blobs/<id>"
                  className="w-full px-3 py-2 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 focus:ring-2 focus:ring-red-500 focus:outline-none text-xs"
                />
              </div>

              {primaryDisplayImage && (
                <div className="flex items-center gap-3 p-3 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-800/40">
                  <img
                    src={primaryDisplayImage}
                    alt="Preview"
                    className="w-14 h-14 rounded-lg object-cover bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-700"
                    onError={(e) => {
                      (e.target as HTMLElement).style.display = "none";
                    }}
                  />
                  <div className="text-xs text-zinc-500 truncate">
                    <span className="font-medium text-zinc-700 dark:text-zinc-300 block">
                      Preview
                    </span>
                    <span className="truncate block font-mono text-[11px]">
                      {primaryDisplayImage}
                    </span>
                  </div>
                </div>
              )}

              {/* Upload Dropzone */}
              <input
                type="file"
                ref={fileInputRef}
                accept="image/png,image/jpeg,image/webp"
                className="hidden"
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  if (file) handleFileUpload(file);
                }}
              />
              <div
                onClick={() => fileInputRef.current?.click()}
                className="border-2 border-dashed border-zinc-300 dark:border-zinc-700 rounded-xl p-4 text-center hover:border-red-500 transition-colors cursor-pointer bg-zinc-50 dark:bg-zinc-800/30"
              >
                {isUploading ? (
                  <div className="flex flex-col items-center justify-center py-2 text-zinc-500">
                    <Loader2 size={24} className="animate-spin text-red-500 mb-1" />
                    <span className="text-xs">Uploading image...</span>
                  </div>
                ) : (
                  <>
                    <Upload size={24} className="mx-auto text-zinc-400 mb-1.5" />
                    <p className="text-xs text-zinc-600 dark:text-zinc-400 font-medium">
                      Click to upload photo or drag & drop
                    </p>
                    <p className="text-[11px] text-zinc-400">
                      PNG, JPG, WebP up to 5MB (stored via /api/blobs)
                    </p>
                  </>
                )}
              </div>
            </div>

            {/* Reference Links */}
            <div className="space-y-4 pt-4 border-t border-zinc-200 dark:border-zinc-800">
              <div className="flex items-center justify-between">
                <h3 className="font-semibold text-xs text-zinc-400 uppercase tracking-wider">
                  Reference Links
                </h3>
                <button
                  type="button"
                  onClick={handleAddLink}
                  className="inline-flex items-center gap-1 text-xs font-semibold text-red-600 dark:text-red-400 hover:underline cursor-pointer"
                >
                  <Plus size={14} /> Add Link
                </button>
              </div>

              {referenceLinks.length === 0 ? (
                <p className="text-xs text-zinc-400 italic">No reference links added.</p>
              ) : (
                <div className="space-y-2">
                  {referenceLinks.map((link, idx) => (
                    <div key={idx} className="flex items-center gap-2">
                      <input
                        type="text"
                        value={link.label}
                        onChange={(e) => handleLinkChange(idx, "label", e.target.value)}
                        placeholder="Label (e.g. Official Manual)"
                        className="w-1/3 px-3 py-1.5 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 text-xs text-zinc-900 dark:text-zinc-100"
                      />
                      <input
                        type="url"
                        value={link.url}
                        onChange={(e) => handleLinkChange(idx, "url", e.target.value)}
                        placeholder="https://example.com/spec-sheet"
                        className="flex-1 px-3 py-1.5 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 text-xs text-zinc-900 dark:text-zinc-100"
                      />
                      <button
                        type="button"
                        onClick={() => handleRemoveLink(idx)}
                        className="p-1.5 text-zinc-400 hover:text-red-500 rounded-lg hover:bg-zinc-100 dark:hover:bg-zinc-800"
                      >
                        <Trash2 size={14} />
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Modal Footer */}
          <div className="px-6 py-3.5 border-t border-zinc-200 dark:border-zinc-800 flex items-center justify-between bg-zinc-50 dark:bg-zinc-900/50">
            <span className="text-xs text-zinc-500">* Required fields</span>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={onClose}
                disabled={isSubmitting}
                className="px-4 py-2 rounded-lg border border-zinc-300 dark:border-zinc-700 text-xs font-medium hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors cursor-pointer"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={isSubmitting}
                className="inline-flex items-center gap-1.5 px-5 py-2 rounded-lg bg-red-600 hover:bg-red-700 text-white text-xs font-semibold shadow-xs transition-colors cursor-pointer disabled:opacity-50"
              >
                {isSubmitting && <Loader2 size={14} className="animate-spin" />}
                {isClone
                  ? `Create ${singularLabel}`
                  : isEditing
                    ? "Update Component"
                    : "Save Component"}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  );
}
