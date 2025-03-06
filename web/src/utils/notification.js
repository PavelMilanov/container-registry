import toast from "solid-toast";

export function showToast(message, type = "info") {
  const backgroundColor = type === "error" ? "#dc3545" : "#1e3c72";
  toast(message, {
    style: {
      "background-color": backgroundColor,
      color: "white",
    },
    className: "notification",
  });
}
