import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { TransportProvider } from "@connectrpc/connect-query";
import type { Transport } from "@connectrpc/connect";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { transport as defaultTransport } from "./lib/api";
import { Layout } from "./components/Layout";
import { CategoryPage } from "./pages/CategoryPage";
import { CollectionPage } from "./pages/CollectionPage";
import { ProductPage } from "./pages/ProductPage";
import { HomePage } from "./pages/HomePage";
import { BuildProfilePage } from "./pages/BuildProfilePage";

const defaultQueryClient = new QueryClient();

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        <Route index element={<HomePage />} />
        <Route path="builds" element={<HomePage />} />
        <Route path="builds/:buildId" element={<BuildProfilePage />} />
        <Route path="components/:categoryId" element={<CategoryPage />} />
        <Route path="components/:categoryId/:collectionId" element={<CollectionPage />} />
        <Route path="components/:categoryId/:collectionId/:productId" element={<ProductPage />} />
      </Route>
    </Routes>
  );
}

export interface AppProps {
  transport?: Transport;
  queryClient?: QueryClient;
}

export function App({
  transport = defaultTransport,
  queryClient = defaultQueryClient,
}: AppProps = {}) {
  return (
    <TransportProvider transport={transport}>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <AppRoutes />
        </BrowserRouter>
      </QueryClientProvider>
    </TransportProvider>
  );
}

export default App;
