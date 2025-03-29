import { createSignal, onMount, lazy } from "solid-js";
import { A, useNavigate, useParams, useLocation } from "@solidjs/router";
import axios from "axios";
import { showToast } from "./utils/notification";
import Breadcrumb from "./utils/Breadcrumb";

const RepoTable = lazy(() => import("./utils/RepoTable"));
const Delete = lazy(() => import("./modal/Delete"));

const API_URL = window.API_URL;

export default function Repo() {
  const navigate = useNavigate();
  const location = useLocation();
  const params = useParams();
  const [imageList, setImageList] = createSignal([]);
  const [repo, setRepo] = createSignal("");
  const [isModalDeleteOpen, setModalDeleteOpen] = createSignal(false);

  const openDeleteModal = (item) => {
    setModalDeleteOpen(true);
    setRepo(item);
  };

  const closeDeleteModal = () => setModalDeleteOpen(false);
  const submitDelete = async () => {
    setModalDeleteOpen(false);
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.delete(
        API_URL + `/api/registry/${params.name}/${repo()}`,
        { headers: headers },
      );
      if (response.status == 202) {
        showToast("Репозиторий удален!");
      }
      await getRepo();
    } catch (error) {
      if (error.response.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      } else {
        console.error(error);
        showToast("Ошибка!", "error");
      }
    }
  };

  async function getRepo() {
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.get(
        API_URL + `/api/registry/${params.name}`,
        {
          headers: headers,
        },
      );
      setImageList(response.data.data); // в ответе приходит массив "data"
    } catch (error) {
      if (error.response?.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      } else {
        console.error(error);
        showToast("Ошибка!", "error");
      }
    }
  }

  onMount(async () => {
    await getRepo();
  });
  return (
    <div class="container">
      <Breadcrumb path={location.pathname} />
      <div class="card">
        <Delete
          isOpen={isModalDeleteOpen()}
          message={"Образы Docker репозитория будут удалены!"}
          onClose={closeDeleteModal}
          onSubmit={submitDelete}
        />
        <RepoTable items={imageList()} />
      </div>
    </div>
  );
}
