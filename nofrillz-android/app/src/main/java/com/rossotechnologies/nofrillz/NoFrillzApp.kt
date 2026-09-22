package com.rossotechnologies.nofrillz

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.navigation.NavHostController
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.rossotechnologies.nofrillz.core.ApiException
import com.rossotechnologies.nofrillz.core.AppContainer
import com.rossotechnologies.nofrillz.core.AuthInputValidator
import com.rossotechnologies.nofrillz.core.CursorPage
import com.rossotechnologies.nofrillz.core.Post
import com.rossotechnologies.nofrillz.core.User
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.time.Duration
import java.time.Instant

private fun NavHostController.navigateTopLevel(route: String) {
    navigate(route) {
        popUpTo("feed") { saveState = true }
        launchSingleTop = true
        restoreState = true
    }
}

@Composable
fun NoFrillzApp() {
    val context = LocalContext.current
    val container = remember { AppContainer(context.applicationContext) }
    val sessionState by container.sessionStore.sessionState.collectAsState()

    if (sessionState.isAuthenticated) {
        AuthenticatedGraph(container = container)
    } else {
        AuthGraph(container = container)
    }
}

@Composable
private fun AuthGraph(container: AppContainer) {
    val navController = rememberNavController()

    NavHost(navController = navController, startDestination = "auth_landing") {
        composable("auth_landing") {
            AuthLandingScreen(
                onLogin = { navController.navigate("login") },
                onSignup = { navController.navigate("signup") },
            )
        }

        composable("login") {
            LoginScreen(
                onBack = { navController.popBackStack() },
                onLogin = { email, password ->
                    container.api.login(email, password).also {
                        container.sessionStore.saveSession(
                            token = it.token,
                            refreshToken = it.refreshToken,
                            user = it.user,
                            userId = it.userId,
                        )
                    }
                },
            )
        }

        composable("signup") {
            SignupScreen(
                onBack = { navController.popBackStack() },
                onSignup = { first, last, username, email, password, about ->
                    container.api.signup(first, last, username, email, password, about).also {
                        container.sessionStore.saveSession(
                            token = it.token,
                            refreshToken = it.refreshToken,
                            user = it.user,
                            userId = it.userId,
                        )
                    }
                },
            )
        }
    }
}

@Composable
private fun AuthLandingScreen(onLogin: () -> Unit, onSignup: () -> Unit) {
    val subtitles = listOf(
        "Minimal social. Max signal.",
        "Follow people, not noise.",
        "Share quickly, read clearly.",
    )
    var subtitleIndex by remember { mutableStateOf(0) }

    LaunchedEffect(Unit) {
        while (true) {
            delay(5000)
            subtitleIndex = (subtitleIndex + 1) % subtitles.size
        }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text("NoFrillz", style = MaterialTheme.typography.headlineLarge, fontWeight = FontWeight.Bold)
        Spacer(Modifier.height(10.dp))
        Text(
            subtitles[subtitleIndex],
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.height(32.dp))
        Button(onClick = onLogin, modifier = Modifier.fillMaxWidth()) {
            Text("Log in")
        }
        Spacer(Modifier.height(12.dp))
        OutlinedButton(onClick = onSignup, modifier = Modifier.fillMaxWidth()) {
            Text("Create account")
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun LoginScreen(
    onBack: () -> Unit,
    onLogin: suspend (email: String, password: String) -> Unit,
) {
    var email by rememberSaveable { mutableStateOf("") }
    var password by rememberSaveable { mutableStateOf("") }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Log in") },
                navigationIcon = { TextButton(onClick = onBack) { Text("Back") } },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(24.dp),
            verticalArrangement = Arrangement.Center,
        ) {
            OutlinedTextField(
                value = email,
                onValueChange = { email = it },
                label = { Text("Email") },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email),
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            Spacer(Modifier.height(12.dp))
            OutlinedTextField(
                value = password,
                onValueChange = { password = it },
                label = { Text("Password") },
                visualTransformation = PasswordVisualTransformation(),
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            Spacer(Modifier.height(20.dp))
            Button(
                onClick = {
                    error = null
                    val trimmedEmail = email.trim()
                    if (trimmedEmail.isEmpty() || password.isEmpty()) {
                        error = "Enter your email and password."
                        return@Button
                    }
                    if (!AuthInputValidator.isValidEmail(trimmedEmail)) {
                        error = "Enter a valid email."
                        return@Button
                    }
                    scope.launch {
                        loading = true
                        val result = runCatching {
                            withContext(Dispatchers.IO) { onLogin(trimmedEmail, password) }
                        }
                        loading = false
                        error = result.exceptionOrNull()?.message
                    }
                },
                modifier = Modifier.fillMaxWidth(),
                enabled = !loading,
            ) {
                if (loading) {
                    CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Text("Log in")
                }
            }
            if (!error.isNullOrBlank()) {
                Spacer(Modifier.height(12.dp))
                Text(error!!, color = MaterialTheme.colorScheme.error)
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun SignupScreen(
    onBack: () -> Unit,
    onSignup: suspend (
        firstName: String,
        lastName: String,
        username: String,
        email: String,
        password: String,
        about: String?,
    ) -> Unit,
) {
    var firstName by rememberSaveable { mutableStateOf("") }
    var lastName by rememberSaveable { mutableStateOf("") }
    var username by rememberSaveable { mutableStateOf("") }
    var email by rememberSaveable { mutableStateOf("") }
    var password by rememberSaveable { mutableStateOf("") }
    var about by rememberSaveable { mutableStateOf("") }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Create account") },
                navigationIcon = { TextButton(onClick = onBack) { Text("Back") } },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(24.dp)
                .verticalScroll(rememberScrollState()),
        ) {
            OutlinedTextField(firstName, { firstName = it }, label = { Text("First name") }, modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(10.dp))
            OutlinedTextField(lastName, { lastName = it }, label = { Text("Last name") }, modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(10.dp))
            OutlinedTextField(username, { username = it }, label = { Text("Username") }, modifier = Modifier.fillMaxWidth(), singleLine = true)
            Spacer(Modifier.height(10.dp))
            OutlinedTextField(email, { email = it }, label = { Text("Email") }, modifier = Modifier.fillMaxWidth(), singleLine = true)
            Spacer(Modifier.height(10.dp))
            OutlinedTextField(
                password,
                { password = it },
                label = { Text("Password") },
                visualTransformation = PasswordVisualTransformation(),
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            Spacer(Modifier.height(10.dp))
            OutlinedTextField(
                value = about,
                onValueChange = { about = it.take(100) },
                label = { Text("About") },
                modifier = Modifier.fillMaxWidth(),
                minLines = 3,
                maxLines = 5,
            )
            Spacer(Modifier.height(4.dp))
            Text("${about.length}/100", color = MaterialTheme.colorScheme.onSurfaceVariant)
            Spacer(Modifier.height(16.dp))
            Button(
                onClick = {
                    error = null
                    val f = firstName.trim()
                    val l = lastName.trim()
                    val u = username.trim()
                    val e = email.trim()
                    val p = password
                    if (f.isEmpty() || l.isEmpty() || u.isEmpty() || e.isEmpty() || p.isEmpty()) {
                        error = "Fill in all required fields."
                        return@Button
                    }
                    if (!AuthInputValidator.isValidUsername(u)) {
                        error = "Username must be 3-32 characters (letters, numbers, underscore)."
                        return@Button
                    }
                    if (!AuthInputValidator.isValidEmail(e)) {
                        error = "Enter a valid email."
                        return@Button
                    }
                    scope.launch {
                        loading = true
                        val result = runCatching {
                            withContext(Dispatchers.IO) {
                                onSignup(f, l, u, e, p, about.trim().ifBlank { null })
                            }
                        }
                        loading = false
                        error = result.exceptionOrNull()?.message
                    }
                },
                enabled = !loading,
                modifier = Modifier.fillMaxWidth(),
            ) {
                if (loading) {
                    CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Text("Create account")
                }
            }
            if (!error.isNullOrBlank()) {
                Spacer(Modifier.height(12.dp))
                Text(error!!, color = MaterialTheme.colorScheme.error)
            }
        }
    }
}

@Composable
private fun AuthenticatedGraph(container: AppContainer) {
    val navController = rememberNavController()
    val sessionState by container.sessionStore.sessionState.collectAsState()

    NavHost(navController = navController, startDestination = "feed") {
        composable("feed") {
            FeedScreen(
                api = container.api,
                onOpenSearch = { navController.navigateTopLevel("search") },
                onOpenCreatePost = { navController.navigate("create_post") },
                onOpenProfile = {
                    val userId = sessionState.user?.id ?: return@FeedScreen
                    navController.navigateTopLevel("profile/$userId/1")
                },
                onOpenPost = { postId -> navController.navigate("post/$postId") },
                onOpenAuthor = { userId -> navController.navigate("profile/$userId/0") },
            )
        }

        composable("search") {
            SearchScreen(
                api = container.api,
                currentUserId = sessionState.user?.id,
                onOpenFeed = { navController.navigateTopLevel("feed") },
                onOpenProfileTab = {
                    val userId = sessionState.user?.id ?: return@SearchScreen
                    navController.navigateTopLevel("profile/$userId/1")
                },
                onOpenProfile = { userId -> navController.navigate("profile/$userId/0") },
            )
        }

        composable("create_post") {
            CreatePostScreen(
                api = container.api,
                onBack = { navController.popBackStack() },
                onCreated = {
                    navController.popBackStack()
                    navController.navigate("feed") {
                        popUpTo("feed") { inclusive = true }
                    }
                },
            )
        }

        composable(
            route = "post/{postId}",
            arguments = listOf(navArgument("postId") { type = NavType.StringType }),
        ) { entry ->
            val postId = entry.arguments?.getString("postId").orEmpty()
            PostDetailScreen(api = container.api, postId = postId, onBack = { navController.popBackStack() })
        }

        composable(
            route = "profile/{userId}/{isSelf}",
            arguments = listOf(
                navArgument("userId") { type = NavType.StringType },
                navArgument("isSelf") { type = NavType.IntType },
            ),
        ) { entry ->
            val userId = entry.arguments?.getString("userId").orEmpty()
            val isSelf = entry.arguments?.getInt("isSelf") == 1
            ProfileScreen(
                api = container.api,
                sessionStore = container.sessionStore,
                userId = userId,
                isSelf = isSelf,
                onBack = { navController.popBackStack() },
                onOpenConnections = { kind -> navController.navigate("connections/$userId/$kind") },
                onOpenPost = { postId -> navController.navigate("post/$postId") },
                onOpenSettings = { navController.navigate("settings") },
                onOpenFeed = { navController.navigateTopLevel("feed") },
                onOpenSearch = { navController.navigateTopLevel("search") },
                onOpenSelfProfile = {
                    val selfId = sessionState.user?.id ?: return@ProfileScreen
                    navController.navigateTopLevel("profile/$selfId/1")
                },
            )
        }

        composable(
            route = "connections/{userId}/{kind}",
            arguments = listOf(
                navArgument("userId") { type = NavType.StringType },
                navArgument("kind") { type = NavType.StringType },
            ),
        ) { entry ->
            val userId = entry.arguments?.getString("userId").orEmpty()
            val kind = entry.arguments?.getString("kind").orEmpty()
            ConnectionsScreen(
                api = container.api,
                currentUserId = sessionState.user?.id,
                userId = userId,
                kind = kind,
                onBack = { navController.popBackStack() },
                onOpenProfile = { profileId -> navController.navigate("profile/$profileId/0") },
            )
        }

        composable("settings") {
            SettingsScreen(
                api = container.api,
                sessionStore = container.sessionStore,
                onBack = { navController.popBackStack() },
            )
        }
    }
}

@Composable
private fun NoFrillzBottomBar(
    selected: String,
    onFeed: () -> Unit,
    onSearch: () -> Unit,
    onProfile: () -> Unit,
) {
    Column {
        HorizontalDivider(color = MaterialTheme.colorScheme.outline)
        NavigationBar(containerColor = MaterialTheme.colorScheme.background) {
            NavigationBarItem(
                selected = selected == "feed",
                onClick = onFeed,
                icon = { Text("Home", style = MaterialTheme.typography.labelSmall) },
            )
            NavigationBarItem(
                selected = selected == "search",
                onClick = onSearch,
                icon = { Text("Search", style = MaterialTheme.typography.labelSmall) },
            )
            NavigationBarItem(
                selected = selected == "profile",
                onClick = onProfile,
                icon = { Text("Me", style = MaterialTheme.typography.labelSmall) },
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun FeedScreen(
    api: com.rossotechnologies.nofrillz.core.NoFrillzApi,
    onOpenSearch: () -> Unit,
    onOpenCreatePost: () -> Unit,
    onOpenProfile: () -> Unit,
    onOpenPost: (String) -> Unit,
    onOpenAuthor: (String) -> Unit,
) {
    var discover by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val posts = remember { mutableStateListOf<Post>() }
    var nextCursor by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var loadingMore by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    fun load(reset: Boolean) {
        scope.launch {
            if (reset) loading = true else loadingMore = true
            val cursor = if (reset) null else nextCursor
            val result = runCatching {
                withContext(Dispatchers.IO) { api.fetchFeedPage(cursor = cursor, discover = discover) }
            }
            result.onSuccess { page ->
                if (reset) posts.clear()
                posts.addAll(page.items)
                nextCursor = page.nextCursor
                error = null
            }.onFailure {
                error = it.message
            }
            if (reset) loading = false else loadingMore = false
        }
    }

    LaunchedEffect(discover) { load(reset = true) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(if (discover) "Discover" else "Following") },
                actions = {
                    TextButton(enabled = !loading && !loadingMore, onClick = { discover = !discover }) { Text(if (discover) "Following" else "Discover") }
                    TextButton(enabled = !loading && !loadingMore, onClick = { load(true) }) { Text("Refresh") }
                },
            )
        },
        bottomBar = {
            NoFrillzBottomBar(
                selected = "feed",
                onFeed = {},
                onSearch = onOpenSearch,
                onProfile = onOpenProfile,
            )
        },
        floatingActionButton = {
            FloatingActionButton(onClick = onOpenCreatePost) {
                Text("+")
            }
        },
    ) { padding ->
        when {
            loading -> Box(Modifier.fillMaxSize().padding(padding), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
            else -> {
                Column(Modifier.fillMaxSize().padding(padding)) {
                    if (!error.isNullOrBlank()) {
                        Text(
                            text = error.orEmpty(),
                            color = MaterialTheme.colorScheme.error,
                            modifier = Modifier.padding(12.dp),
                        )
                    }
                    LazyColumn(modifier = Modifier.fillMaxSize()) {
                        items(posts, key = { it.id }) { post ->
                            PostCard(
                                post = post,
                                showAuthor = true,
                                onAuthorTap = { onOpenAuthor(post.authorId) },
                                onPostTap = { onOpenPost(post.id) },
                                onLikeTap = {
                                    val index = posts.indexOfFirst { it.id == post.id }
                                    if (index >= 0) {
                                        val nextLiked = !posts[index].liked
                                        posts[index] = posts[index].withLiked(nextLiked)
                                        scope.launch(Dispatchers.IO) {
                                            runCatching {
                                                if (nextLiked) api.likePost(post.id) else api.unlikePost(post.id)
                                            }
                                        }
                                    }
                                },
                            )
                        }
                        item {
                            if (nextCursor != null) {
                                TextButton(
                                    onClick = { if (!loadingMore) load(false) },
                                    modifier = Modifier.fillMaxWidth(),
                                ) {
                                    if (loadingMore) CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                                    else Text("Load more")
                                }
                            } else if (posts.isNotEmpty()) {
                                Text(
                                    text = "No more posts",
                                    modifier = Modifier.padding(16.dp),
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun SearchScreen(
    api: com.rossotechnologies.nofrillz.core.NoFrillzApi,
    currentUserId: String?,
    onOpenFeed: () -> Unit,
    onOpenProfileTab: () -> Unit,
    onOpenProfile: (String) -> Unit,
) {
    val scope = rememberCoroutineScope()
    var query by rememberSaveable { mutableStateOf("") }
    val results = remember { mutableStateListOf<User>() }
    val following = remember { mutableStateListOf<String>() }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(query) {
        val term = query.trim()
        delay(500)
        if (term.isEmpty()) {
            results.clear()
            return@LaunchedEffect
        }
        loading = true
        val result = runCatching {
            withContext(Dispatchers.IO) { api.searchUsers(term) }
        }
        loading = false
        result.onSuccess { users ->
            results.clear()
            results.addAll(users.filter { it.id != currentUserId })
            following.clear()
            following.addAll(users.filter { it.isFollowing }.map { it.id })
            error = null
        }.onFailure {
            error = it.message
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Search") },
            )
        },
        bottomBar = {
            NoFrillzBottomBar(
                selected = "search",
                onFeed = onOpenFeed,
                onSearch = {},
                onProfile = onOpenProfileTab,
            )
        },
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding).padding(horizontal = 16.dp)) {
            OutlinedTextField(
                value = query,
                onValueChange = { query = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text("Find users") },
                singleLine = true,
            )
            Spacer(Modifier.height(10.dp))
            if (loading) CircularProgressIndicator()
            if (!error.isNullOrBlank()) {
                Text(error.orEmpty(), color = MaterialTheme.colorScheme.error)
            }
            if (results.isEmpty() && !loading) {
                Spacer(Modifier.height(20.dp))
                Text("No users found", color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            LazyColumn {
                items(results, key = { it.id }) { user ->
                    UserRow(
                        user = user,
                        isFollowing = following.contains(user.id) || user.isFollowing,
                        followEnabled = user.id != currentUserId,
                        onFollow = {
                            if (following.contains(user.id)) return@UserRow
                            scope.launch {
                                runCatching { withContext(Dispatchers.IO) { api.followUser(user.id) } }
                                    .onSuccess { following.add(user.id) }
                                    .onFailure { error = it.message }
                            }
                        },
                        onOpen = { onOpenProfile(user.id) },
                    )
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun CreatePostScreen(
    api: com.rossotechnologies.nofrillz.core.NoFrillzApi,
    onBack: () -> Unit,
    onCreated: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    var content by rememberSaveable { mutableStateOf("") }
    var posting by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Create post") },
                navigationIcon = { TextButton(onClick = onBack) { Text("Back") } },
            )
        },
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding).padding(16.dp)) {
            OutlinedTextField(
                value = content,
                onValueChange = { content = it },
                label = { Text("What is happening?") },
                minLines = 7,
                maxLines = 12,
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(12.dp))
            Button(
                onClick = {
                    val trimmed = content.trim()
                    if (trimmed.isEmpty()) {
                        error = "Post content is required"
                        return@Button
                    }
                    scope.launch {
                        posting = true
                        val result = runCatching {
                            withContext(Dispatchers.IO) { api.createPost(trimmed) }
                        }
                        posting = false
                        result.onSuccess {
                            content = ""
                            onCreated()
                        }.onFailure {
                            error = it.message
                        }
                    }
                },
                enabled = !posting,
                modifier = Modifier.fillMaxWidth(),
            ) {
                if (posting) CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                else Text("Post")
            }
            Spacer(Modifier.height(8.dp))
            Text("Posts are public to people who can see your profile.", color = MaterialTheme.colorScheme.onSurfaceVariant)
            if (!error.isNullOrBlank()) {
                Spacer(Modifier.height(10.dp))
                Text(error.orEmpty(), color = MaterialTheme.colorScheme.error)
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun PostDetailScreen(
    api: com.rossotechnologies.nofrillz.core.NoFrillzApi,
    postId: String,
    onBack: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    var post by remember { mutableStateOf<Post?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }

    fun refresh() {
        scope.launch {
            loading = true
            val result = runCatching {
                withContext(Dispatchers.IO) { api.fetchPost(postId) }
            }
            loading = false
            result.onSuccess {
                post = it
                error = null
            }.onFailure {
                error = it.message
            }
        }
    }

    LaunchedEffect(postId) { refresh() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Post") },
                navigationIcon = { TextButton(onClick = onBack) { Text("Back") } },
                actions = { TextButton(onClick = { refresh() }) { Text("Refresh") } },
            )
        },
    ) { padding ->
        Box(Modifier.fillMaxSize().padding(padding).padding(16.dp)) {
            when {
                loading -> CircularProgressIndicator(Modifier.align(Alignment.Center))
                post != null -> {
                    Column {
                        Text(post!!.authorDisplayName, fontWeight = FontWeight.Bold)
                        val aiMarker = if (post!!.isAi) " • ai" else ""
                        Text(
                            (post!!.createdAt?.let(::formatPostDateShort) ?: "") + aiMarker,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                        Spacer(Modifier.height(16.dp))
                        Text(post!!.content, style = MaterialTheme.typography.bodyLarge)
                        Spacer(Modifier.height(18.dp))
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            TextButton(onClick = {
                                val current = post ?: return@TextButton
                                val nextLiked = !current.liked
                                post = current.withLiked(nextLiked)
                                scope.launch(Dispatchers.IO) {
                                    runCatching {
                                        if (nextLiked) api.likePost(current.id) else api.unlikePost(current.id)
                                    }
                                }
                            }) {
                                Text(
                                    if (post!!.liked) "Liked" else "Like",
                                    color = if (post!!.liked) MaterialTheme.colorScheme.tertiary else MaterialTheme.colorScheme.onSurface,
                                )
                            }
                        }
                    }
                }
                else -> Text(error ?: "Post unavailable", color = MaterialTheme.colorScheme.error)
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ProfileScreen(
    api: com.rossotechnologies.nofrillz.core.NoFrillzApi,
    sessionStore: com.rossotechnologies.nofrillz.core.SessionStore,
    userId: String,
    isSelf: Boolean,
    onBack: () -> Unit,
    onOpenConnections: (String) -> Unit,
    onOpenPost: (String) -> Unit,
    onOpenSettings: () -> Unit,
    onOpenFeed: () -> Unit,
    onOpenSearch: () -> Unit,
    onOpenSelfProfile: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    var user by remember { mutableStateOf<User?>(null) }
    val posts = remember { mutableStateListOf<Post>() }
    var nextCursor by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var loadingMore by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    fun load(reset: Boolean) {
        scope.launch {
            if (reset) loading = true else loadingMore = true
            val result = runCatching {
                withContext(Dispatchers.IO) {
                    val profile = if (isSelf) api.fetchCurrentUser() else api.fetchUser(userId)
                    val page = api.fetchPostsPage(userId = userId, cursor = if (reset) null else nextCursor)
                    profile to page
                }
            }
            if (reset) loading = false else loadingMore = false
            result.onSuccess { (profile, page) ->
                user = profile
                if (isSelf) sessionStore.updateUser(profile)
                if (reset) posts.clear()
                posts.addAll(page.items)
                nextCursor = page.nextCursor ?: if (page.items.size >= 20) page.items.lastOrNull()?.id else null
                error = null
            }.onFailure {
                error = it.message
            }
        }
    }

    LaunchedEffect(userId) { load(true) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(if (isSelf) "Profile" else "User") },
                navigationIcon = {
                    if (!isSelf) {
                        TextButton(onClick = onBack) { Text("Back") }
                    }
                },
                actions = {
                    if (isSelf) {
                        TextButton(onClick = onOpenSettings) { Text("Settings") }
                    }
                },
            )
        },
        bottomBar = {
            if (isSelf) {
                NoFrillzBottomBar(
                    selected = "profile",
                    onFeed = onOpenFeed,
                    onSearch = onOpenSearch,
                    onProfile = onOpenSelfProfile,
                )
            }
        },
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding)) {
            if (loading) {
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
                return@Column
            }
            if (!error.isNullOrBlank()) {
                Text(error.orEmpty(), color = MaterialTheme.colorScheme.error, modifier = Modifier.padding(12.dp))
            }
            val resolvedUser = user
            if (resolvedUser != null) {
                ProfileHeader(
                    user = resolvedUser,
                    showRelation = !isSelf,
                    onOpenFollowers = { onOpenConnections("followers") },
                    onOpenFollowing = { onOpenConnections("following") },
                )
            }
            LazyColumn(modifier = Modifier.fillMaxSize()) {
                items(posts, key = { it.id }) { post ->
                    PostCard(
                        post = post,
                        showAuthor = false,
                        onAuthorTap = {},
                        onPostTap = { onOpenPost(post.id) },
                        onLikeTap = {
                            val index = posts.indexOfFirst { it.id == post.id }
                            if (index >= 0) {
                                val nextLiked = !posts[index].liked
                                posts[index] = posts[index].withLiked(nextLiked)
                                scope.launch(Dispatchers.IO) {
                                    runCatching {
                                        if (nextLiked) api.likePost(post.id) else api.unlikePost(post.id)
                                    }
                                }
                            }
                        },
                    )
                }
                item {
                    if (nextCursor != null) {
                        TextButton(onClick = { if (!loadingMore) load(false) }, modifier = Modifier.fillMaxWidth()) {
                            if (loadingMore) CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                            else Text("Load more")
                        }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ConnectionsScreen(
    api: com.rossotechnologies.nofrillz.core.NoFrillzApi,
    currentUserId: String?,
    userId: String,
    kind: String,
    onBack: () -> Unit,
    onOpenProfile: (String) -> Unit,
) {
    val scope = rememberCoroutineScope()
    val users = remember { mutableStateListOf<User>() }
    val following = remember { mutableStateListOf<String>() }
    var nextCursor by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var loadingMore by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    fun load(reset: Boolean) {
        scope.launch {
            if (reset) loading = true else loadingMore = true
            val result = runCatching {
                withContext(Dispatchers.IO) {
                    if (kind == "followers") api.fetchFollowersPage(userId, cursor = if (reset) null else nextCursor)
                    else api.fetchFollowingPage(userId, cursor = if (reset) null else nextCursor)
                }
            }
            if (reset) loading = false else loadingMore = false
            result.onSuccess { page: CursorPage<User> ->
                if (reset) users.clear()
                users.addAll(page.items)
                page.items.filter { it.isFollowing }.forEach { if (!following.contains(it.id)) following.add(it.id) }
                nextCursor = page.nextCursor
                error = null
            }.onFailure {
                error = it.message
            }
        }
    }

    LaunchedEffect(userId, kind) { load(true) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(if (kind == "followers") "Followers" else "Following") },
                navigationIcon = { TextButton(onClick = onBack) { Text("Back") } },
            )
        },
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding)) {
            if (loading) {
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
                return@Column
            }
            if (!error.isNullOrBlank()) {
                Text(error.orEmpty(), color = MaterialTheme.colorScheme.error, modifier = Modifier.padding(12.dp))
            }
            if (users.isEmpty()) {
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    Text("No users yet", color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                return@Column
            }
            LazyColumn {
                items(users, key = { it.id }) { user ->
                    UserRow(
                        user = user,
                        isFollowing = following.contains(user.id) || user.isFollowing,
                        followEnabled = user.id != currentUserId,
                        onFollow = {
                            if (user.id == currentUserId || following.contains(user.id)) return@UserRow
                            scope.launch {
                                runCatching { withContext(Dispatchers.IO) { api.followUser(user.id) } }
                                    .onSuccess { following.add(user.id) }
                                    .onFailure { error = it.message }
                            }
                        },
                        onOpen = { onOpenProfile(user.id) },
                    )
                }
                item {
                    if (nextCursor != null) {
                        TextButton(onClick = { if (!loadingMore) load(false) }, modifier = Modifier.fillMaxWidth()) {
                            if (loadingMore) CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                            else Text("Load more")
                        }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun SettingsScreen(
    api: com.rossotechnologies.nofrillz.core.NoFrillzApi,
    sessionStore: com.rossotechnologies.nofrillz.core.SessionStore,
    onBack: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Settings") },
                navigationIcon = { TextButton(onClick = onBack) { Text("Back") } },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp),
            verticalArrangement = Arrangement.Bottom,
        ) {
            Button(
                onClick = {
                    scope.launch {
                        loading = true
                        runCatching { withContext(Dispatchers.IO) { api.signOutCurrentSession() } }
                        sessionStore.clear()
                        loading = false
                    }
                },
                colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error),
                enabled = !loading,
                modifier = Modifier.fillMaxWidth(),
            ) {
                if (loading) CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                else Text("Log out", color = MaterialTheme.colorScheme.onError)
            }
            if (!error.isNullOrBlank()) {
                Spacer(Modifier.height(8.dp))
                Text(error.orEmpty(), color = MaterialTheme.colorScheme.error)
            }
        }
    }
}

@Composable
private fun ProfileHeader(
    user: User,
    showRelation: Boolean,
    onOpenFollowers: () -> Unit,
    onOpenFollowing: () -> Unit,
) {
    Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 12.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Avatar(initials = user.initials)
            Spacer(Modifier.width(12.dp))
            Column(Modifier.weight(1f)) {
                Text(user.displayName, style = MaterialTheme.typography.headlineSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text("@${user.username}", color = MaterialTheme.colorScheme.onSurfaceVariant)
                if (showRelation) {
                    val relation = buildList {
                        if (user.isFollowing) add("you follow")
                        if (user.isFollower) add("follows you")
                    }.joinToString("  •  ")
                    if (relation.isNotBlank()) {
                        Text(relation, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
            }
        }
        Spacer(Modifier.height(10.dp))
        Text(user.about ?: "No bio yet")
        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            OutlinedButton(onClick = onOpenFollowers, modifier = Modifier.weight(1f)) {
                Text("Followers ${user.followerCount}")
            }
            OutlinedButton(onClick = onOpenFollowing, modifier = Modifier.weight(1f)) {
                Text("Following ${user.followingCount}")
            }
            OutlinedButton(onClick = {}, modifier = Modifier.weight(1f)) {
                Text("Posts ${user.postCount}")
            }
        }
    }
}

@Composable
private fun UserRow(
    user: User,
    isFollowing: Boolean,
    followEnabled: Boolean,
    onFollow: () -> Unit,
    onOpen: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onOpen)
            .padding(horizontal = 12.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Avatar(initials = user.initials)
        Spacer(Modifier.width(12.dp))
        Column(Modifier.weight(1f)) {
            Text(user.displayName, fontWeight = FontWeight.SemiBold)
            Text("@${user.username}", color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        if (followEnabled) {
            if (isFollowing) {
                OutlinedButton(onClick = {}, enabled = false) { Text("Following") }
            } else {
                Button(onClick = onFollow) { Text("Follow") }
            }
        }
    }
}

@Composable
private fun PostCard(
    post: Post,
    showAuthor: Boolean,
    onAuthorTap: () -> Unit,
    onPostTap: () -> Unit,
    onLikeTap: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onPostTap)
            .padding(horizontal = 16.dp, vertical = 18.dp),
    ) {
        if (showAuthor) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable(onClick = onAuthorTap),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Avatar(initials = buildInitials(post.authorFirstName, post.authorLastName, post.authorUsername))
                Spacer(Modifier.width(10.dp))
                Column {
                    val time = post.createdAt?.let(::formatPostDateShort)
                    val aiMarker = if (post.isAi) " • ai" else ""
                    Text(
                        text = listOfNotNull(post.authorDisplayName, time).joinToString(" • ") + aiMarker,
                        fontWeight = FontWeight.SemiBold,
                        style = MaterialTheme.typography.titleMedium,
                    )
                    Text("@${post.authorUsername}", color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
            Spacer(Modifier.height(8.dp))
        }
        Text(
            text = post.content,
            style = MaterialTheme.typography.bodyLarge,
        )
        Spacer(Modifier.height(8.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            TextButton(onClick = onLikeTap) {
                Text(
                    text = if (post.liked) "Liked" else "Like",
                    color = if (post.liked) MaterialTheme.colorScheme.tertiary else MaterialTheme.colorScheme.onSurface,
                )
            }
            if (!showAuthor) {
                post.createdAt?.let {
                    Text(formatPostDateShort(it), color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
        }
        HorizontalDivider(color = MaterialTheme.colorScheme.outline)
    }
}

@Composable
private fun Avatar(initials: String) {
    Box(
        modifier = Modifier
            .size(40.dp)
            .background(MaterialTheme.colorScheme.surfaceVariant, CircleShape),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            initials.lowercase(),
            color = MaterialTheme.colorScheme.onSurface,
            fontWeight = FontWeight.SemiBold,
        )
    }
}

private fun buildInitials(first: String?, last: String?, username: String): String {
    val f = first.orEmpty().trim().take(1)
    val l = last.orEmpty().trim().take(1)
    return ("$f$l").ifBlank { username.take(1) }.lowercase()
}

private fun formatPostDateShort(date: Instant): String {
    val age = Duration.between(date, Instant.now()).seconds.coerceAtLeast(0)
    return when {
        age < 60 -> "now"
        age < 3600 -> "${(age / 60).coerceAtLeast(1)}m"
        age < 86400 -> "${(age / 3600).coerceAtLeast(1)}h"
        age < 604800 -> "${(age / 86400).coerceAtLeast(1)}d"
        age < 2628000 -> "${(age / 604800).coerceAtLeast(1)}w"
        age < 31536000 -> "${(age / 2628000).coerceAtLeast(1)}mo"
        else -> "${(age / 31536000).coerceAtLeast(1)}y"
    }
}

private fun Throwable.userMessage(): String {
    return when (this) {
        is ApiException -> message ?: "Request failed"
        else -> message ?: "Something went wrong"
    }
}
