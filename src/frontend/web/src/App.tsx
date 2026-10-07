import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { TransportProvider } from "@connectrpc/connect-query";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { transport } from "./lib/api";
import { Layout } from "./components/Layout";
import { CategoryPage } from "./pages/CategoryPage";
import { CollectionPage } from "./pages/CollectionPage";
import { ProductPage } from "./pages/ProductPage";

import { HomePage } from "./pages/HomePage";

const queryClient = new QueryClient();

function App() {
  return (
    <TransportProvider transport={transport}>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <Routes>
            <Route path="/" element={<Layout />}>
              <Route index element={<HomePage />} />
              <Route path="components/:categoryId" element={<CategoryPage />} />
              <Route path="components/:categoryId/:collectionId" element={<CollectionPage />} />
              <Route path="components/:categoryId/:collectionId/:productId" element={<ProductPage />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </QueryClientProvider>
    </TransportProvider>
  );
}

export default App;
