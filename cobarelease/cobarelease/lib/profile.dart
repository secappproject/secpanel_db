import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:shimmer/shimmer.dart';
import 'package:secpanel/helpers/db_helper.dart';
import 'package:secpanel/models/approles.dart';
import 'package:secpanel/models/company.dart';
import 'package:secpanel/models/companyaccount.dart'; // Pastikan ini diimpor
import 'package:secpanel/theme/colors.dart';

class ProfileScreen extends StatefulWidget {
  const ProfileScreen({super.key});

  @override
  State<ProfileScreen> createState() => _ProfileScreenState();
}

class _ProfileScreenState extends State<ProfileScreen>
    with TickerProviderStateMixin {
  Company? _currentCompany;
  String? _loggedInUsername; // <-- TAMBAHKAN VARIABEL INI
  bool _isLoading = true;

  Map<String, List<Map<String, dynamic>>> _groupedUsers = {};
  List<String> _companyGroupNames = [];
  List<Map<String, dynamic>> _allOtherUsers = [];
  List<Map<String, dynamic>> _colleagues = [];
  bool _isLoadingUsers = true;

  TabController? _tabController;

  @override
  void initState() {
    super.initState();
    _loadInitialData();
  }

  @override
  void dispose() {
    _tabController?.dispose();
    super.dispose();
  }

  Future<void> _loadInitialData() async {
    if (!mounted) return;
    setState(() => _isLoading = true);
    await _loadCompanyData();
    if (_currentCompany != null && _loggedInUsername != null) {
      if (_currentCompany!.role == AppRole.admin) {
        await _loadAndGroupAllUsers();
      } else {
        await _loadColleaguesAccounts();
      }
    }
    if (mounted) setState(() => _isLoading = false);
  }

  Future<void> _loadCompanyData() async {
    final prefs = await SharedPreferences.getInstance();
    final username = prefs.getString('loggedInUsername');
    if (username != null) {
      final company = await DatabaseHelper.instance.getCompanyByUsername(
        username,
      );
      if (mounted) {
        setState(() {
          _currentCompany = company;
          _loggedInUsername = username; // <-- SIMPAN USERNAME DI SINI
        });
      }
    }
  }

  Future<void> _loadAndGroupAllUsers() async {
    if (mounted) setState(() => _isLoadingUsers = true);

    final allUsersFromDb = await DatabaseHelper.instance
        .getAllUserAccountsForDisplay();

    final Map<String, List<Map<String, dynamic>>> grouped = {};
    for (var user in allUsersFromDb) {
      final companyName = user['company_name'] as String;
      (grouped[companyName] ??= []).add(user);
    }

    if (mounted) {
      setState(() {
        _allOtherUsers = allUsersFromDb;
        _groupedUsers = grouped;
        _companyGroupNames = grouped.keys.toList()..sort();
        _isLoadingUsers = false;

        final tabLength = _companyGroupNames.isNotEmpty
            ? _companyGroupNames.length + 1
            : 0;

        if (_tabController?.length != tabLength) {
          _tabController?.dispose();
          _tabController = tabLength > 0
              ? TabController(length: tabLength, vsync: this)
              : null;
        }
        _tabController?.addListener(() {
          if (mounted) setState(() {});
        });
      });
    }
  }

  Future<void> _loadColleaguesAccounts() async {
    if (!mounted || _currentCompany == null || _loggedInUsername == null)
      return;

    setState(() => _isLoadingUsers = true);
    final colleaguesAccounts = await DatabaseHelper.instance
        .getColleagueAccountsForDisplay(
          _currentCompany!.name,
          _loggedInUsername!,
        );
    if (mounted) {
      setState(() {
        _colleagues = colleaguesAccounts;
        _isLoadingUsers = false;
      });
    }
  }

  Future<void> _logout() async {
    // 1. Show confirmation dialog and wait for user's choice
    final bool? confirm = await showModalBottomSheet<bool>(
      context: context,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => const _LogoutConfirmationSheet(),
    );

    // 2. Proceed only if the user confirmed (confirm is true)
    if (confirm == true) {
      final prefs = await SharedPreferences.getInstance();
      await prefs.clear();

      if (mounted) {
        // 3. Show a success message
        _showSuccessSnackBar("Anda telah berhasil keluar.");

        // 4. Wait a moment for the user to see the message
        await Future.delayed(const Duration(milliseconds: 800));

        if (mounted) {
          // 5. Navigate to the login screen
          Navigator.pushNamedAndRemoveUntil(
            context,
            '/login',
            (route) => false,
          );
        }
      }
    }
  }

  void _showSuccessSnackBar(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        backgroundColor: Colors.green, // Or AppColors.schneiderGreen
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  void _showCompanyFormBottomSheet({Company? company}) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (context) {
        // Jika mengedit, ID yang diteruskan ke _CompanyFormBottomSheet
        // adalah username akun, bukan ID perusahaan. Ini sudah sesuai
        // dengan implementasi _CompanyFormBottomSheet saat ini.
        return Scaffold(
          backgroundColor: Colors.transparent,
          body: _CompanyFormBottomSheet(
            company: company,
            onSave: _loadInitialData,
          ),
        );
      },
    );
  }

  String _formatRole(AppRole role) {
    return role.name[0].toUpperCase() + role.name.substring(1);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.white,
      body: SafeArea(
        child: _isLoading
            ? const _ProfileSkeleton()
            : _currentCompany == null ||
                  _loggedInUsername ==
                      null // <-- PASTIKAN JUGA _loggedInUsername TIDAK NULL
            ? const Center(child: Text("Gagal memuat data pengguna."))
            : _buildProfileContent(),
      ),
    );
  }

  Widget _buildProfileContent() {
    return RefreshIndicator(
      onRefresh: _loadInitialData,
      color: AppColors.schneiderGreen,
      child: SingleChildScrollView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.all(20.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              "Profil",
              style: TextStyle(
                color: AppColors.black,
                fontSize: 24,
                fontWeight: FontWeight.w500,
              ),
            ),
            const SizedBox(height: 24),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Image.asset('assets/images/user.png', height: 44, width: 44),
                InkWell(
                  onTap: _logout,
                  child: Container(
                    padding: const EdgeInsets.all(8),
                    decoration: const BoxDecoration(
                      border: Border(
                        left: BorderSide(color: AppColors.red, width: 1.5),
                      ),
                    ),
                    child: Row(
                      children: [
                        Image.asset(
                          'assets/images/logout.png',
                          height: 24,
                          width: 24,
                        ),
                        const SizedBox(width: 4),
                        const Text(
                          "Keluar",
                          style: TextStyle(
                            color: AppColors.red,
                            fontSize: 12,
                            fontWeight: FontWeight.w400,
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 24),
            // Sekarang ini akan menampilkan nama perusahaan, misal "DSM" atau "Abacus"
            Text(
              _currentCompany!.name,
              style: const TextStyle(fontSize: 20, fontWeight: FontWeight.w400),
            ),
            const SizedBox(height: 24),
            const Text(
              "Username",
              style: TextStyle(fontSize: 12, fontWeight: FontWeight.w400),
            ),
            const SizedBox(height: 8),
            Container(
              padding: const EdgeInsets.all(12),
              width: double.infinity,
              decoration: const BoxDecoration(
                color: AppColors.grayLight,
                borderRadius: BorderRadius.all(Radius.circular(8)),
              ),
              child: Text(
                _loggedInUsername!, // <-- TAMPILKAN USERNAME ASLI DI SINI
                style: const TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w300,
                ),
              ),
            ),
            const SizedBox(height: 12),
            const Text(
              "Role",
              style: TextStyle(fontSize: 12, fontWeight: FontWeight.w400),
            ),
            const SizedBox(height: 8),
            Container(
              padding: const EdgeInsets.all(12),
              width: double.infinity,
              decoration: const BoxDecoration(
                color: AppColors.grayLight,
                borderRadius: BorderRadius.all(Radius.circular(8)),
              ),
              child: Text(
                _formatRole(_currentCompany!.role),
                style: const TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w300,
                ),
              ),
            ),
            _buildUserListSection(),
          ],
        ),
      ),
    );
  }

  Widget _buildUserListSection() {
    bool isAdmin = _currentCompany!.role == AppRole.admin;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const SizedBox(height: 32),
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(
              isAdmin ? "Pengguna" : "Rekan di Perusahaan Anda",
              style: const TextStyle(
                color: AppColors.black,
                fontSize: 20,
                fontWeight: FontWeight.w500,
              ),
            ),
            if (isAdmin)
              OutlinedButton.icon(
                icon: const Icon(
                  Icons.add,
                  size: 18,
                  color: AppColors.schneiderGreen,
                ),
                label: const Text(
                  "Tambah",
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w400,
                    color: AppColors.black,
                  ),
                ),
                onPressed: () => _showCompanyFormBottomSheet(),
                style: OutlinedButton.styleFrom(
                  side: const BorderSide(color: AppColors.grayLight),
                ),
              ),
          ],
        ),
        const SizedBox(height: 16),
        _isLoadingUsers
            ? const Center(
                child: CircularProgressIndicator(
                  color: AppColors.schneiderGreen,
                ),
              )
            : isAdmin
            ? _buildAdminUserListView()
            : _buildStandardUserListView(),
      ],
    );
  }

  Widget _buildAdminUserListView() {
    if (_allOtherUsers.isEmpty || _tabController == null) {
      return const Center(
        child: Padding(
          padding: EdgeInsets.symmetric(vertical: 24.0),
          child: Text("Belum ada pengguna lain."),
        ),
      );
    }

    final allTab = Tab(text: "All (${_allOtherUsers.length})");
    final groupTabs = _companyGroupNames
        .map((name) => Tab(text: "$name (${_groupedUsers[name]!.length})"))
        .toList();

    final allTabView = _buildUserListView(_allOtherUsers);
    final groupTabViews = _companyGroupNames.map((groupName) {
      return _buildUserListView(_groupedUsers[groupName]!);
    }).toList();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        TabBar(
          controller: _tabController,
          isScrollable: true,
          labelColor: AppColors.black,
          unselectedLabelColor: AppColors.gray,
          indicatorColor: AppColors.schneiderGreen,
          tabAlignment: TabAlignment.start,
          labelStyle: const TextStyle(
            fontSize: 11,
            fontWeight: FontWeight.w400,
            fontFamily: 'Lexend',
          ),
          unselectedLabelStyle: const TextStyle(
            fontSize: 11,
            fontWeight: FontWeight.w400,
            fontFamily: 'Lexend',
          ),
          tabs: [allTab, ...groupTabs],
        ),
        IndexedStack(
          index: _tabController!.index,
          children: [allTabView, ...groupTabViews]
              .asMap()
              .map(
                (index, view) => MapEntry(
                  index,
                  Visibility(
                    visible: _tabController!.index == index,
                    maintainState: true,
                    child: view,
                  ),
                ),
              )
              .values
              .toList(),
        ),
      ],
    );
  }

  Widget _buildStandardUserListView() {
    if (_colleagues.isEmpty) {
      return const Center(
        child: Padding(
          padding: EdgeInsets.symmetric(vertical: 24.0),
          child: Text("Tidak ada rekan lain di perusahaan ini."),
        ),
      );
    }
    return _buildUserListView(_colleagues, isAdminView: false);
  }

  Widget _buildUserListView(
    List<Map<String, dynamic>> users, {
    bool isAdminView = true,
  }) {
    return ListView.separated(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      padding: const EdgeInsets.only(top: 16),
      itemCount: users.length,
      separatorBuilder: (context, index) => const SizedBox(height: 16),
      itemBuilder: (context, index) {
        final user = users[index];
        return _UserCard(
          userData: user,
          onEdit: isAdminView
              ? () {
                  final companyForEdit = Company(
                    id: user['username'] as String,
                    name: user['company_name'] as String,
                    role: AppRole.values.firstWhere(
                      (e) => e.name == user['role'],
                      orElse: () => AppRole.k3,
                    ),
                  );
                  _showCompanyFormBottomSheet(company: companyForEdit);
                }
              : null,
        );
      },
    );
  }
}

class _UserCard extends StatelessWidget {
  final Map<String, dynamic> userData;
  final VoidCallback? onEdit;

  const _UserCard({required this.userData, this.onEdit});

  String _formatRole(String roleName) {
    if (roleName.isEmpty) return '';
    final formatted = roleName[0].toUpperCase() + roleName.substring(1);
    return formatted;
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        borderRadius: const BorderRadius.all(Radius.circular(8)),
        border: Border.all(width: 1, color: AppColors.grayLight),
      ),
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 12, 12, 8),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.center,
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        userData['username'] as String,
                        style: const TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        _formatRole(userData['role'] as String),
                        style: const TextStyle(
                          fontSize: 12,
                          color: AppColors.gray,
                          fontWeight: FontWeight.w400,
                        ),
                      ),
                    ],
                  ),
                ),
                if (onEdit != null)
                  InkWell(
                    onTap: onEdit,
                    borderRadius: BorderRadius.circular(8),
                    child: Container(
                      padding: const EdgeInsets.all(8),
                      decoration: BoxDecoration(
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(
                          color: AppColors.grayLight,
                          width: 1,
                        ),
                      ),
                      child: Image.asset(
                        'assets/images/edit-green.png',
                        height: 20,
                      ),
                    ),
                  ),
              ],
            ),
          ),
          const Divider(height: 1, color: AppColors.grayLight),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            width: double.infinity,
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                const Text(
                  "Perusahaan",
                  style: TextStyle(
                    color: AppColors.gray,
                    fontWeight: FontWeight.w300,
                    fontSize: 12,
                  ),
                ),
                Text(
                  userData['company_name'] as String,
                  style: const TextStyle(
                    color: AppColors.gray,
                    fontWeight: FontWeight.w400,
                    fontSize: 12,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _CompanyFormBottomSheet extends StatefulWidget {
  final Company? company;
  final VoidCallback onSave;
  const _CompanyFormBottomSheet({this.company, required this.onSave});
  @override
  State<_CompanyFormBottomSheet> createState() =>
      _CompanyFormBottomSheetState();
}

class _CompanyFormBottomSheetState extends State<_CompanyFormBottomSheet> {
  final _formKey = GlobalKey<FormState>();
  late TextEditingController _idController;
  late TextEditingController _passwordController;
  AppRole _selectedRole = AppRole.k3;
  bool _isEditing = false;
  List<Map<String, String>> _companiesData = [];
  String? _selectedCompanyName;
  bool _isLoadingCompanies = true;

  @override
  void initState() {
    super.initState();
    _isEditing = widget.company != null;
    _idController = TextEditingController(text: widget.company?.id ?? '');
    _passwordController = TextEditingController();
    if (_isEditing) {
      _selectedCompanyName = widget.company!.name;
      _selectedRole = widget.company!.role;
    }
    _loadCompaniesData();
  }

  Future<void> _loadCompaniesData() async {
    setState(() => _isLoadingCompanies = true);
    final data = await DatabaseHelper.instance.getUniqueCompanyDataForForm();
    if (mounted) {
      setState(() {
        _companiesData = data;
        _isLoadingCompanies = false;
      });
    }
  }

  @override
  void dispose() {
    _idController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _saveUser() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }

    if (_selectedCompanyName == null || _selectedCompanyName!.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text("Nama Perusahaan wajib dipilih"),
          backgroundColor: AppColors.red,
        ),
      );
      return;
    }

    final username = _idController.text.trim();
    if (!_isEditing) {
      final bool isTaken = await DatabaseHelper.instance.isUsernameTaken(
        username,
      );
      if (isTaken) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
              content: Text(
                "Username ini sudah digunakan. Silakan gunakan username lain.",
              ),
              backgroundColor: AppColors.red,
            ),
          );
        }
        return;
      }
    }

    final companyName = _selectedCompanyName!;
    final password = _passwordController.text.trim();

    try {
      if (_isEditing) {
        final companyToUpdate = Company(
          id: username,
          name: companyName,
          role: _selectedRole,
        );
        await DatabaseHelper.instance.updateCompanyAndAccount(
          companyToUpdate,
          newPassword: password.isNotEmpty ? password : null,
        );
      } else {
        Company? existingCompany = await DatabaseHelper.instance
            .getCompanyByName(companyName);
        String companyId;

        if (existingCompany != null) {
          companyId = existingCompany.id;
          if (existingCompany.role != _selectedRole) {}
        } else {
          companyId = companyName.toLowerCase().replaceAll(RegExp(r'\s+'), '_');
          final newCompany = Company(
            id: companyId,
            name: companyName,
            role: _selectedRole,
          );
          existingCompany = newCompany;
        }

        final account = CompanyAccount(
          username: username,
          password: password,
          companyId: companyId,
        );
        await DatabaseHelper.instance.insertCompanyWithAccount(
          existingCompany,
          account,
        );
      }

      _showSuccessAndPop();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text("Terjadi kesalahan: ${e.toString()}"),
            backgroundColor: AppColors.red,
          ),
        );
      }
    }
  }

  void _showSuccessAndPop() {
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            _isEditing
                ? 'Pengguna berhasil diperbarui'
                : 'Pengguna berhasil ditambahkan',
          ),
          backgroundColor: AppColors.schneiderGreen,
        ),
      );
      widget.onSave();
      Navigator.pop(context);
    }
  }

  Future<void> _deleteUser() async {
    final confirm = await showModalBottomSheet<bool>(
      context: context,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => _DeleteConfirmationSheet(userName: widget.company!.id),
    );

    if (confirm == true) {
      try {
        await DatabaseHelper.instance.deleteCompanyAccount(widget.company!.id);
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
              content: Text('Pengguna berhasil dihapus'),
              backgroundColor: AppColors.schneiderGreen,
            ),
          );
          widget.onSave();
          Navigator.pop(context);
        }
      } catch (e) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text("Gagal menghapus: $e"),
              backgroundColor: AppColors.red,
            ),
          );
        }
      }
    }
  }

  Future<void> _showAddNewCompanySheet() async {
    final newCompanyData = await showModalBottomSheet<Map<String, dynamic>>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => const _AddNewCompanyRoleSheet(),
    );

    if (newCompanyData != null) {
      final String newName = newCompanyData['name'];
      final AppRole newRole = newCompanyData['role'];

      if (!_companiesData.any((c) => c['name'] == newName)) {
        setState(() {
          _companiesData.add({'name': newName, 'role': newRole.name});
        });
      }
      setState(() {
        _selectedCompanyName = newName;
        _selectedRole = newRole;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      padding: EdgeInsets.fromLTRB(
        20,
        16,
        20,
        MediaQuery.of(context).viewInsets.bottom + 16,
      ),
      child: Form(
        key: _formKey,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Center(
              child: Container(
                height: 5,
                width: 40,
                decoration: BoxDecoration(
                  color: AppColors.grayLight,
                  borderRadius: BorderRadius.circular(100),
                ),
              ),
            ),
            const SizedBox(height: 24),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(
                  _isEditing ? 'Edit Pengguna' : 'Tambah Pengguna Baru',
                  style: const TextStyle(
                    fontSize: 20,
                    fontWeight: FontWeight.w500,
                  ),
                ),
                if (_isEditing)
                  IconButton(
                    icon: const Icon(
                      Icons.delete_outline,
                      color: AppColors.red,
                    ),
                    onPressed: _deleteUser,
                  ),
              ],
            ),
            const SizedBox(height: 24),
            _buildCompanyNameSelector(),
            const SizedBox(height: 16),
            _buildTextField(
              controller: _idController,
              label: 'Username',
              enabled: !_isEditing,
            ),
            const SizedBox(height: 16),
            _buildTextField(
              controller: _passwordController,
              label: _isEditing ? 'Password Baru (Opsional)' : 'Password',
            ),
            const SizedBox(height: 32),
            _buildActionButtons(context: context, onSave: _saveUser),
          ],
        ),
      ),
    );
  }

  Widget _buildCompanyNameSelector() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          "Perusahaan",
          style: TextStyle(fontSize: 14, fontWeight: FontWeight.w400),
        ),
        const SizedBox(height: 12),
        _isLoadingCompanies
            ? const Center(
                child: CircularProgressIndicator(
                  color: AppColors.schneiderGreen,
                ),
              )
            : Wrap(
                spacing: 8,
                runSpacing: 12,
                children: [
                  ..._companiesData.map((data) {
                    final name = data['name']!;
                    final role = data['role']!;
                    return _buildCompanyOptionButton(
                      name: name,
                      role: role,
                      selected: _selectedCompanyName == name,
                      onTap: () {
                        setState(() {
                          _selectedCompanyName = name;
                          _selectedRole = AppRole.values.firstWhere(
                            (e) => e.name == role,
                          );
                        });
                      },
                    );
                  }),
                  _buildOtherButton(onTap: _showAddNewCompanySheet),
                ],
              ),
      ],
    );
  }

  Widget _buildCompanyOptionButton({
    required String name,
    required String role,
    required bool selected,
    required VoidCallback onTap,
  }) {
    final Color borderColor = selected
        ? AppColors.schneiderGreen
        : AppColors.grayLight;
    final Color color = selected
        ? AppColors.schneiderGreen.withOpacity(0.08)
        : Colors.white;

    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        decoration: BoxDecoration(
          color: color,
          border: Border.all(color: borderColor),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            Text(
              name,
              style: const TextStyle(
                fontWeight: FontWeight.w400,
                fontSize: 12,
                color: AppColors.black,
              ),
            ),
            const SizedBox(width: 8),
            Chip(
              label: Text(
                role[0].toUpperCase() + role.substring(1),
                style: const TextStyle(fontSize: 10, color: AppColors.gray),
              ),
              backgroundColor: AppColors.grayLight,
              padding: EdgeInsets.zero,
              labelPadding: const EdgeInsets.symmetric(horizontal: 6),
              visualDensity: VisualDensity.compact,
              side: BorderSide.none,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildOtherButton({required VoidCallback onTap}) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          color: Colors.white,
          border: Border.all(color: AppColors.grayLight),
          borderRadius: BorderRadius.circular(8),
        ),
        child: const Text(
          "Lainnya...",
          style: TextStyle(
            fontWeight: FontWeight.w400,
            fontSize: 12,
            color: AppColors.gray,
          ),
        ),
      ),
    );
  }

  Widget _buildTextField({
    required TextEditingController controller,
    required String label,
    bool enabled = true,
  }) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w400),
        ),
        const SizedBox(height: 8),
        TextFormField(
          cursorColor: AppColors.schneiderGreen,
          controller: controller,
          enabled: enabled,
          obscureText: label.toLowerCase().contains('password'),
          style: const TextStyle(
            fontSize: 12,
            fontWeight: FontWeight.w300,
            color: AppColors.black,
          ),
          validator: (v) {
            if (label.contains('Opsional') && (v == null || v.isEmpty)) {
              return null;
            }
            if (v == null || v.isEmpty) return 'Field ini tidak boleh kosong';
            return null;
          },
          decoration: InputDecoration(
            fillColor: enabled ? AppColors.white : AppColors.grayLight,
            filled: true,
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 16,
              vertical: 12,
            ),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: AppColors.grayLight),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: AppColors.grayLight),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: AppColors.schneiderGreen),
            ),
            disabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: AppColors.grayLight),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildActionButtons({
    required BuildContext context,
    required VoidCallback onSave,
  }) {
    return Row(
      children: [
        Expanded(
          child: OutlinedButton(
            onPressed: () => Navigator.pop(context),
            style: OutlinedButton.styleFrom(
              padding: const EdgeInsets.symmetric(vertical: 16),
              side: const BorderSide(color: AppColors.schneiderGreen),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(6),
              ),
            ),
            child: const Text(
              "Batal",
              style: TextStyle(color: AppColors.schneiderGreen, fontSize: 12),
            ),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: ElevatedButton(
            onPressed: onSave,
            style: ElevatedButton.styleFrom(
              padding: const EdgeInsets.symmetric(vertical: 16),
              backgroundColor: AppColors.schneiderGreen,
              foregroundColor: Colors.white,
              elevation: 0,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(6),
              ),
            ),
            child: const Text("Simpan", style: TextStyle(fontSize: 12)),
          ),
        ),
      ],
    );
  }
}

class _AddNewCompanyRoleSheet extends StatefulWidget {
  const _AddNewCompanyRoleSheet();
  @override
  State<_AddNewCompanyRoleSheet> createState() =>
      _AddNewCompanyRoleSheetState();
}

class _AddNewCompanyRoleSheetState extends State<_AddNewCompanyRoleSheet> {
  final _formKey = GlobalKey<FormState>();
  final _nameController = TextEditingController();
  AppRole _selectedRole = AppRole.k3;

  @override
  void dispose() {
    _nameController.dispose();
    super.dispose();
  }

  void _save() {
    if (_formKey.currentState!.validate()) {
      Navigator.pop(context, {
        'name': _nameController.text.trim(),
        'role': _selectedRole,
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      padding: EdgeInsets.fromLTRB(
        20,
        16,
        20,
        MediaQuery.of(context).viewInsets.bottom + 16,
      ),
      child: Form(
        key: _formKey,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Center(
              child: Container(
                height: 5,
                width: 40,
                decoration: BoxDecoration(
                  color: AppColors.grayLight,
                  borderRadius: BorderRadius.circular(100),
                ),
              ),
            ),
            const SizedBox(height: 24),
            const Text(
              "Tambah Perusahaan Baru",
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
            ),
            const SizedBox(height: 24),
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text(
                  'Company',
                  style: TextStyle(fontSize: 14, fontWeight: FontWeight.w400),
                ),
                const SizedBox(height: 12),
                TextFormField(
                  cursorColor: AppColors.schneiderGreen,
                  controller: _nameController,
                  autofocus: true,
                  style: const TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w300,
                    color: AppColors.black,
                  ),
                  validator: (v) => (v == null || v.isEmpty)
                      ? 'Nama tidak boleh kosong'
                      : null,
                  decoration: InputDecoration(
                    fillColor: AppColors.white,
                    filled: true,
                    contentPadding: const EdgeInsets.symmetric(
                      horizontal: 16,
                      vertical: 12,
                    ),
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide: const BorderSide(color: AppColors.grayLight),
                    ),
                    enabledBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide: const BorderSide(color: AppColors.grayLight),
                    ),
                    focusedBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide: const BorderSide(
                        color: AppColors.schneiderGreen,
                      ),
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 24),
            _buildRoleSelector(),
            const SizedBox(height: 32),
            _buildActionButtons(),
          ],
        ),
      ),
    );
  }

  Widget _buildRoleSelector() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          'Role',
          style: TextStyle(fontSize: 14, fontWeight: FontWeight.w400),
        ),
        const SizedBox(height: 12),
        Wrap(
          spacing: 8,
          runSpacing: 12,
          children: AppRole.values.map((role) {
            if (role == AppRole.admin) return const SizedBox.shrink();
            return _buildOptionButton(
              label: role.name[0].toUpperCase() + role.name.substring(1),
              selected: _selectedRole == role,
              onTap: () => setState(() => _selectedRole = role),
            );
          }).toList(),
        ),
      ],
    );
  }

  Widget _buildOptionButton({
    required String label,
    required bool selected,
    required VoidCallback onTap,
  }) {
    final Color borderColor = selected
        ? AppColors.schneiderGreen
        : AppColors.grayLight;
    final Color color = selected
        ? AppColors.schneiderGreen.withOpacity(0.08)
        : Colors.white;

    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          color: color,
          border: Border.all(color: borderColor),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Text(
          label,
          style: const TextStyle(
            fontWeight: FontWeight.w400,
            fontSize: 12,
            color: AppColors.black,
          ),
        ),
      ),
    );
  }

  Widget _buildActionButtons() {
    return Row(
      children: [
        Expanded(
          child: OutlinedButton(
            onPressed: () => Navigator.pop(context),
            style: OutlinedButton.styleFrom(
              padding: const EdgeInsets.symmetric(vertical: 16),
              side: const BorderSide(color: AppColors.schneiderGreen),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(6),
              ),
            ),
            child: const Text(
              "Batal",
              style: TextStyle(color: AppColors.schneiderGreen, fontSize: 12),
            ),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: ElevatedButton(
            onPressed: _save,
            style: ElevatedButton.styleFrom(
              padding: const EdgeInsets.symmetric(vertical: 16),
              backgroundColor: AppColors.schneiderGreen,
              foregroundColor: Colors.white,
              elevation: 0,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(6),
              ),
            ),
            child: const Text("Simpan", style: TextStyle(fontSize: 12)),
          ),
        ),
      ],
    );
  }
}

class _DeleteConfirmationSheet extends StatelessWidget {
  final String userName;
  const _DeleteConfirmationSheet({required this.userName});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 16, 20, 24),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Center(
            child: Container(
              height: 5,
              width: 40,
              decoration: BoxDecoration(
                color: AppColors.grayLight,
                borderRadius: BorderRadius.circular(100),
              ),
            ),
          ),
          const SizedBox(height: 24),
          const Text(
            "Hapus Pengguna?",
            style: TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
          ),
          const SizedBox(height: 8),
          Text(
            "Anda yakin ingin menghapus pengguna \"$userName\"? Tindakan ini tidak dapat diurungkan.",
            style: const TextStyle(color: AppColors.gray, fontSize: 14),
          ),
          const SizedBox(height: 32),
          Row(
            children: [
              Expanded(
                child: OutlinedButton(
                  onPressed: () => Navigator.pop(context, false),
                  style: OutlinedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(vertical: 16),
                    side: const BorderSide(color: AppColors.schneiderGreen),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(6),
                    ),
                  ),
                  child: const Text(
                    "Batal",
                    style: TextStyle(
                      color: AppColors.schneiderGreen,
                      fontSize: 12,
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: ElevatedButton(
                  onPressed: () => Navigator.pop(context, true),
                  style: ElevatedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(vertical: 16),
                    backgroundColor: AppColors.red,
                    foregroundColor: Colors.white,
                    elevation: 0,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(6),
                    ),
                  ),
                  child: const Text(
                    "Ya, Hapus",
                    style: TextStyle(fontSize: 12),
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _ProfileSkeleton extends StatelessWidget {
  const _ProfileSkeleton();
  @override
  Widget build(BuildContext context) {
    return Shimmer.fromColors(
      baseColor: Colors.grey[200]!,
      highlightColor: Colors.grey[100]!,
      child: SingleChildScrollView(
        physics: const NeverScrollableScrollPhysics(),
        padding: const EdgeInsets.all(20.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              width: 100,
              height: 28,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(8),
              ),
            ),
            const SizedBox(height: 24),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Container(
                  width: 44,
                  height: 44,
                  decoration: const BoxDecoration(
                    color: Colors.white,
                    shape: BoxShape.circle,
                  ),
                ),
                Container(
                  width: 80,
                  height: 30,
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(8),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 24),
            Container(
              width: 200,
              height: 24,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(8),
              ),
            ),
            const SizedBox(height: 24),
            Container(
              width: 80,
              height: 14,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(8),
              ),
            ),
            const SizedBox(height: 8),
            Container(
              width: double.infinity,
              height: 48,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(8),
              ),
            ),
            const SizedBox(height: 12),
            Container(
              width: 50,
              height: 14,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(8),
              ),
            ),
            const SizedBox(height: 8),
            Container(
              width: double.infinity,
              height: 48,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(8),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _LogoutConfirmationSheet extends StatelessWidget {
  const _LogoutConfirmationSheet();

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 16, 20, 24),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Center(
            child: Container(
              height: 5,
              width: 40,
              decoration: BoxDecoration(
                color: AppColors.grayLight,
                borderRadius: BorderRadius.circular(100),
              ),
            ),
          ),
          const SizedBox(height: 24),
          const Text(
            "Konfirmasi Keluar",
            style: TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
          ),
          const SizedBox(height: 8),
          const Text(
            "Anda yakin ingin keluar dari akun Anda?",
            style: TextStyle(color: AppColors.gray, fontSize: 14),
          ),
          const SizedBox(height: 32),
          Row(
            children: [
              Expanded(
                child: OutlinedButton(
                  onPressed: () =>
                      Navigator.pop(context, false), // Return false
                  style: OutlinedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(vertical: 16),
                    side: const BorderSide(color: AppColors.schneiderGreen),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(6),
                    ),
                  ),
                  child: const Text(
                    "Batal",
                    style: TextStyle(
                      color: AppColors.schneiderGreen,
                      fontSize: 12,
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: ElevatedButton(
                  onPressed: () => Navigator.pop(context, true), // Return true
                  style: ElevatedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(vertical: 16),
                    backgroundColor: AppColors.red,
                    foregroundColor: Colors.white,
                    elevation: 0,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(6),
                    ),
                  ),
                  child: const Text(
                    "Ya, Keluar",
                    style: TextStyle(fontSize: 12),
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
