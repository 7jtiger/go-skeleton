<script lang="ts">
  import Router from 'svelte-spa-router'
  import Home from './pages/public/Home.svelte'
  import Solutions from './pages/public/Solutions.svelte'
  import Chat from './pages/public/Chat.svelte'
  import About from './pages/public/About.svelte'
  import Contact from './pages/public/Contact.svelte'
  import Login from './pages/admin/Login.svelte'
  import Dashboard from './pages/admin/Dashboard.svelte'
  import ContentEditor from './pages/admin/ContentEditor.svelte'
  import Inquiries from './pages/admin/Inquiries.svelte'

  const routes = {
    '/': Home,
    '/solutions': Solutions,
    '/chat': Chat,
    '/about': About,
    '/contact': Contact,
    '/admin/login': Login,
    '/admin': Dashboard,
    '/admin/content': ContentEditor,
    '/admin/inquiries': Inquiries,
  }
</script>

<Router {routes} />
