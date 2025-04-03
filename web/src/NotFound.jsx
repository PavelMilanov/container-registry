import { createEffect } from "solid-js";
import { useNavigate } from "@solidjs/router";

export default function NotFound() {
  const navigate = useNavigate();

  createEffect(() => {
    setTimeout(() => navigate("/registry", { replace: true }), 2000);
  });

  return (
    <div>
      <h1>404 - Страница не найдена</h1>
      <p>Перенаправляем на главную через 2 секунды...</p>
    </div>
  );
}
