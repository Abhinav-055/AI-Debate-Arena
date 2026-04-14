import React from "react";
import ReactDOM from "react-dom/client";
import { ApolloProvider } from "@apollo/client";
import { BrowserRouter } from "react-router-dom";
import { Toaster } from "react-hot-toast";
import App from "./App";
import { apolloClient } from "./apollo/client";
import "./styles.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <ApolloProvider client={apolloClient}>
      <BrowserRouter>
        <App />
        <Toaster
          position="top-right"
          toastOptions={{
            style: {
              background: "#1C1C1F",
              color: "#F0F0F0",
              border: "1px solid #2A2A2E",
            },
          }}
        />
      </BrowserRouter>
    </ApolloProvider>
  </React.StrictMode>,
);
