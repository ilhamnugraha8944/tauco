"use client";

import { useEffect } from "react";

export default function ErrorPage({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <section className="error-page">
      <div className="mx-auto w-full max-w-[1280px] px-5 sm:px-8 lg:px-10">
        <p className="eyebrow">Terjadi kendala</p>
        <h1>Halaman belum dapat ditampilkan</h1>
        <p>Coba muat kembali halaman ini.</p>
        <button type="button" className="button-link" onClick={reset}>
          Coba kembali
        </button>
      </div>
    </section>
  );
}
