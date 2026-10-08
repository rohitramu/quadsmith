import React from "react";
import { render, type RenderOptions } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { TransportProvider } from "@connectrpc/connect-query";
import type { Transport } from "@connectrpc/connect";
import { MemoryRouter } from "react-router-dom";
import { createMockTransport, type MockTransportOptions } from "./mocks/transport";
import { AppRoutes } from "../App";

export function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
        staleTime: 0,
      },
    },
  });
}

export interface RenderWithProvidersOptions extends Omit<RenderOptions, "wrapper"> {
  route?: string;
  initialEntries?: string[];
  transport?: Transport;
  transportOptions?: MockTransportOptions;
  queryClient?: QueryClient;
}

export function renderWithProviders(
  ui: React.ReactElement,
  options: RenderWithProvidersOptions = {},
) {
  const {
    route = "/",
    initialEntries = [route],
    transport = createMockTransport(options.transportOptions),
    queryClient = createTestQueryClient(),
    ...renderOptions
  } = options;

  function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <TransportProvider transport={transport}>
        <QueryClientProvider client={queryClient}>
          <MemoryRouter initialEntries={initialEntries}>{children}</MemoryRouter>
        </QueryClientProvider>
      </TransportProvider>
    );
  }

  return {
    user: userEvent.setup(),
    queryClient,
    transport,
    ...render(ui, { wrapper: Wrapper, ...renderOptions }),
  };
}

export interface RenderAppOptions extends Omit<RenderWithProvidersOptions, "route"> {
  initialRoute?: string;
}

export function renderApp(initialRoute: string = "/", options: RenderAppOptions = {}) {
  return renderWithProviders(<AppRoutes />, {
    route: initialRoute,
    ...options,
  });
}

export * from "@testing-library/react";
export { userEvent };
