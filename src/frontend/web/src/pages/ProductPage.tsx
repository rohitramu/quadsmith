import { useParams } from "react-router-dom";
import { useQuery } from "@connectrpc/connect-query";
import { getMotor } from "../gen/quadsmith/motor-MotorService_connectquery";
import { formatStatorSize } from "../lib/format";

export function ProductPage() {
  const { categoryId, collectionId, productId } = useParams();

  const isMotors = collectionId === "motors";
  const { data, isLoading, error } = useQuery(getMotor, { id: productId }, { enabled: isMotors && !!productId });

  if (!isMotors) {
    return <p>Product page for {collectionId} coming soon.</p>;
  }

  if (isLoading) return <p>Loading details...</p>;
  if (error) return <p className="text-red-500">Error: {error.message}</p>;
  if (!data) return <p>Product not found.</p>;

  const m = data;

  return (
    <div className="max-w-3xl">
      <div className="mb-4 text-sm text-zinc-500 dark:text-zinc-400 capitalize flex gap-2">
        <span>{categoryId}</span> &gt; <span>{collectionId?.replace('_', ' ')}</span> &gt; <span>{m.name || m.id}</span>
      </div>
      <div className="mb-8">
        <h1 className="text-3xl font-bold">{m.name || m.id}</h1>
        <p className="text-xl text-zinc-500 dark:text-zinc-400">{m.manufacturer || "Unknown Manufacturer"}</p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="p-4 bg-zinc-50 dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
          <h3 className="text-sm font-semibold text-zinc-500 uppercase">KV Rating</h3>
          <p className="text-2xl mt-1">{m.kv || "N/A"} KV</p>
        </div>
        <div className="p-4 bg-zinc-50 dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
          <h3 className="text-sm font-semibold text-zinc-500 uppercase">Weight</h3>
          <p className="text-2xl mt-1">{m.weightG ? `${m.weightG}g` : "N/A"}</p>
        </div>
        <div className="p-4 bg-zinc-50 dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
          <h3 className="text-sm font-semibold text-zinc-500 uppercase">Stator Size</h3>
          <p className="text-2xl mt-1 font-mono">{formatStatorSize(m.statorDiameterMm, m.statorHeightMm, "N/A")}</p>
        </div>
      </div>
    </div>
  );
}
