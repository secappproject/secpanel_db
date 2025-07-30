// file: lib/components/import_review_screen.dart

import 'package:flutter/material.dart';
import 'package:secpanel/components/import/confirm_import_bottom_sheet.dart';
import 'package:secpanel/components/import/import_progress_dialog.dart';
import 'package:secpanel/helpers/db_helper.dart';
import 'package:secpanel/theme/colors.dart';

class ImportReviewScreen extends StatefulWidget {
  final Map<String, List<Map<String, dynamic>>> initialData;
  const ImportReviewScreen({super.key, required this.initialData});

  @override
  State<ImportReviewScreen> createState() => _ImportReviewScreenState();
}

class _ImportReviewScreenState extends State<ImportReviewScreen> {
  late Map<String, List<Map<String, dynamic>>> _editableData;
  late Map<String, Set<int>> _duplicateRows;
  late Map<String, Set<int>> _brokenRelationRows;
  bool _isLoading = true;

  late Map<String, Set<String>> _existingPrimaryKeys;

  static const Map<String, List<String>> _templateColumns = {
    'companies': ['id', 'name', 'role'],
    'company_accounts': ['username', 'password', 'company_id'],
    'panels': [
      'no_pp',
      'no_panel',
      'no_wbs',
      'percent_progress',
      'start_date',
      'status_busbar',
      'status_component',
      'status_palet',
      'status_corepart',
      'created_by',
      'vendor_id',
      'is_closed',
      'closed_date',
    ],
    'busbars': ['panel_no_pp', 'vendor', 'remarks'],
    'components': ['panel_no_pp', 'vendor'],
    'palet': ['panel_no_pp', 'vendor'],
    'corepart': ['panel_no_pp', 'vendor'],
  };

  final ValueNotifier<double> _progressNotifier = ValueNotifier(0.0);
  final ValueNotifier<String> _statusNotifier = ValueNotifier('');

  @override
  void initState() {
    super.initState();
    _editableData = widget.initialData.map((key, value) {
      return MapEntry(
        key,
        value.map((item) => Map<String, dynamic>.from(item)).toList(),
      );
    });
    _initializeAndValidateData();
  }

  Future<void> _initializeAndValidateData() async {
    if (mounted) setState(() => _isLoading = true);
    await _fetchExistingPrimaryKeys();
    _revalidateOnDataChange();
    if (mounted) setState(() => _isLoading = false);
  }

  Future<void> _fetchExistingPrimaryKeys() async {
    final dbHelper = DatabaseHelper.instance;
    _existingPrimaryKeys = {
      'companies': (await dbHelper.getAllCompanies()).map((c) => c.id).toSet(),
      'company_accounts': (await dbHelper.getAllCompanyAccounts())
          .map((a) => a.username)
          .toSet(),
      'panels': (await dbHelper.getAllPanels()).map((p) => p.noPp).toSet(),
      'busbars': (await dbHelper.getAllBusbars())
          .map((b) => "${b.panelNoPp}_${b.vendor}")
          .toSet(),
      'components': (await dbHelper.getAllComponents())
          .map((c) => "${c.panelNoPp}_${c.vendor}")
          .toSet(),
      'palet': (await dbHelper.getAllPalet())
          .map((c) => "${c.panelNoPp}_${c.vendor}")
          .toSet(),
      'corepart': (await dbHelper.getAllCorepart())
          .map((c) => "${c.panelNoPp}_${c.vendor}")
          .toSet(),
    };
  }

  void _validateDuplicates() {
    _duplicateRows = {};
    const Map<String, String> primaryKeyMapping = {
      'companies': 'id',
      'company_accounts': 'username',
      'panels': 'no_pp',
    };
    for (var entry in primaryKeyMapping.entries) {
      final tableName = entry.key;
      final pkColumn = entry.value;
      if (_editableData.containsKey(tableName) &&
          _editableData[tableName]!.isNotEmpty &&
          (_editableData[tableName]!.first.containsKey(pkColumn))) {
        _duplicateRows.putIfAbsent(tableName, () => <int>{});
        final rows = _editableData[tableName]!;
        final pksInDb = _existingPrimaryKeys[tableName] ?? {};
        final pksInFile = <String>{};
        for (int i = 0; i < rows.length; i++) {
          final pkValue = rows[i][pkColumn]?.toString();
          if (pkValue != null && pkValue.isNotEmpty) {
            if (pksInDb.contains(pkValue) || !pksInFile.add(pkValue)) {
              _duplicateRows[tableName]!.add(i);
            }
          }
        }
      }
    }
    final List<String> compositeKeyTables = [
      'busbars',
      'components',
      'palet',
      'corepart',
    ];
    for (final tableName in compositeKeyTables) {
      if (!_editableData.containsKey(tableName) ||
          _editableData[tableName]!.isEmpty)
        continue;
      _duplicateRows.putIfAbsent(tableName, () => <int>{});
      final rows = _editableData[tableName]!;
      final existingCompositeKeys =
          _existingPrimaryKeys[tableName] ?? <String>{};
      final seenKeysInFile = <String>{};
      for (int i = 0; i < rows.length; i++) {
        final row = rows[i];
        final panelNoPp = row['panel_no_pp']?.toString() ?? '';
        final vendor = row['vendor']?.toString() ?? '';
        if (panelNoPp.isNotEmpty && vendor.isNotEmpty) {
          final compositeKey = "${panelNoPp}_${vendor}";
          if (existingCompositeKeys.contains(compositeKey) ||
              !seenKeysInFile.add(compositeKey)) {
            _duplicateRows[tableName]!.add(i);
          }
        }
      }
    }
  }

  void _validateBrokenRelations() {
    _brokenRelationRows = {};
    final allCompanyIDsInDb = _existingPrimaryKeys['companies'] ?? {};
    final allCompanyIDsInFile =
        _editableData['companies']
            ?.map((row) => row['id']?.toString() ?? '')
            .where((id) => id.isNotEmpty)
            .toSet() ??
        {};
    final allAvailableCompanyIDs = {
      ...allCompanyIDsInDb,
      ...allCompanyIDsInFile,
    };

    final allPanelNoPpsInDb = _existingPrimaryKeys['panels'] ?? {};
    final allPanelNoPpsInFile =
        _editableData['panels']
            ?.map((row) => row['no_pp']?.toString() ?? '')
            .where((id) => id.isNotEmpty)
            .toSet() ??
        {};
    final allAvailablePanelNoPps = {
      ...allPanelNoPpsInDb,
      ...allPanelNoPpsInFile,
    };

    _editableData.forEach((tableName, rows) {
      if (rows.isEmpty) return;
      _brokenRelationRows.putIfAbsent(tableName, () => <int>{});
      for (int i = 0; i < rows.length; i++) {
        final row = rows[i];
        bool isBroken = false;
        switch (tableName) {
          case 'company_accounts':
            final fk = row['company_id']?.toString() ?? '';
            if (fk.isNotEmpty && !allAvailableCompanyIDs.contains(fk)) {
              isBroken = true;
            }
            break;
          case 'panels':
            final createdBy = row['created_by']?.toString() ?? '';
            final vendorId = row['vendor_id']?.toString() ?? '';
            if ((createdBy.isNotEmpty &&
                    !allAvailableCompanyIDs.contains(createdBy)) ||
                (vendorId.isNotEmpty &&
                    !allAvailableCompanyIDs.contains(vendorId))) {
              isBroken = true;
            }
            break;
          case 'busbars':
          case 'components':
          case 'palet':
          case 'corepart':
            final panelFk = row['panel_no_pp']?.toString() ?? '';
            final vendorFk = row['vendor']?.toString() ?? '';
            if ((panelFk.isNotEmpty &&
                    !allAvailablePanelNoPps.contains(panelFk)) ||
                (vendorFk.isNotEmpty &&
                    !allAvailableCompanyIDs.contains(vendorFk))) {
              isBroken = true;
            }
            break;
        }
        if (isBroken) {
          _brokenRelationRows[tableName]!.add(i);
        }
      }
    });
  }

  void _revalidateOnDataChange() {
    setState(() {
      _validateDuplicates();
      _validateBrokenRelations();
    });
  }

  void _addRow(String tableName) {
    setState(() {
      final columns = _editableData[tableName]!.isNotEmpty
          ? _editableData[tableName]!.first.keys.toList()
          : _templateColumns[tableName] ?? <String>[];
      final newRow = {for (var col in columns) col: ''};
      _editableData[tableName]!.add(newRow);
      _revalidateOnDataChange();
    });
  }

  void _deleteRow(String tableName, int index) {
    setState(() {
      _editableData[tableName]!.removeAt(index);
      _revalidateOnDataChange();
    });
  }

  void _deleteColumn(String tableName, String columnName) {
    setState(() {
      for (var row in _editableData[tableName]!) {
        row.remove(columnName);
      }
      _revalidateOnDataChange();
    });
  }

  void _renameColumn(String tableName, String oldName, String newName) {
    if (newName.isNotEmpty && newName != oldName) {
      setState(() {
        for (var row in _editableData[tableName]!) {
          final value = row.remove(oldName);
          row[newName] = value;
        }
        _revalidateOnDataChange();
      });
    }
  }

  void _addNewColumn(String tableName, String newName) {
    if (newName.isNotEmpty) {
      setState(() {
        for (var row in _editableData[tableName]!) {
          row[newName] = '';
        }
        _revalidateOnDataChange();
      });
    }
  }

  Future<void> _saveToDatabase() async {
    final hasDuplicates = _duplicateRows.values.any((s) => s.isNotEmpty);
    if (hasDuplicates) {
      _showErrorSnackBar('Data duplikat tidak bisa disimpan. Harap perbaiki.');
      return;
    }

    final confirm = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => const ConfirmImportBottomSheet(
        title: 'Konfirmasi Impor',
        content:
            'Data yang valid akan ditambahkan ke database. Data yang relasinya belum lengkap dapat dilengkapi nanti. Lanjutkan?',
      ),
    );
    if (confirm != true) return;
    showDialog(
      context: context,
      barrierDismissible: false,
      builder: (_) => ImportProgressDialog(
        progress: _progressNotifier,
        status: _statusNotifier,
      ),
    );
    try {
      await DatabaseHelper.instance.importData(_editableData, (p, m) {
        _progressNotifier.value = p;
        _statusNotifier.value = m;
      });
      if (mounted) {
        Navigator.of(context).pop();
        Navigator.of(context).pop(true);
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Data berhasil diimpor! 🎉'),
            backgroundColor: AppColors.schneiderGreen,
            behavior: SnackBarBehavior.floating,
          ),
        );
      }
    } catch (e) {
      if (mounted) {
        Navigator.of(context).pop();
        _showErrorSnackBar('Gagal menyimpan data: $e');
      }
    }
  }

  void _showErrorSnackBar(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        backgroundColor: AppColors.red,
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    if (_isLoading) {
      return const Scaffold(
        backgroundColor: AppColors.white,
        body: Center(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              CircularProgressIndicator(color: AppColors.schneiderGreen),
              SizedBox(height: 16),
              Text(
                "Memvalidasi data...",
                style: TextStyle(color: AppColors.gray),
              ),
            ],
          ),
        ),
      );
    }
    final tableNames = _editableData.keys.toList();
    return DefaultTabController(
      length: tableNames.length,
      child: Scaffold(
        backgroundColor: AppColors.white,
        appBar: AppBar(
          scrolledUnderElevation: 0,
          backgroundColor: AppColors.white,
          surfaceTintColor: AppColors.white,
          title: const Text(
            'Tinjau Data Impor',
            style: TextStyle(
              color: AppColors.black,
              fontSize: 24,
              fontWeight: FontWeight.w400,
            ),
          ),
          bottom: PreferredSize(
            preferredSize: const Size.fromHeight(50),
            child: Align(
              alignment: Alignment.centerLeft,
              child: TabBar(
                isScrollable: true,
                labelColor: AppColors.black,
                unselectedLabelColor: AppColors.gray,
                indicatorColor: AppColors.schneiderGreen,
                indicatorWeight: 2,
                tabAlignment: TabAlignment.start,
                padding: const EdgeInsets.symmetric(horizontal: 20),
                indicatorSize: TabBarIndicatorSize.label,
                overlayColor: WidgetStateProperty.all(Colors.transparent),
                dividerColor: Colors.transparent,
                labelStyle: const TextStyle(
                  fontWeight: FontWeight.w500,
                  fontFamily: 'Lexend',
                  fontSize: 12,
                ),
                unselectedLabelStyle: const TextStyle(
                  fontWeight: FontWeight.w400,
                  fontFamily: 'Lexend',
                  fontSize: 12,
                ),
                tabs: tableNames.map(_buildTabWithIndicator).toList(),
              ),
            ),
          ),
        ),
        body: TabBarView(
          children: tableNames
              .map((name) => _buildDataTable(name, _editableData[name]!))
              .toList(),
        ),
        bottomNavigationBar: Container(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
          decoration: const BoxDecoration(color: AppColors.white),
          child: ElevatedButton(
            style: ElevatedButton.styleFrom(
              minimumSize: const Size(double.infinity, 52),
              shadowColor: Colors.transparent,
              backgroundColor: AppColors.schneiderGreen,
              foregroundColor: Colors.white,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(6),
              ),
            ),
            onPressed: _saveToDatabase,
            child: const Text(
              'Simpan ke Database',
              style: TextStyle(
                fontFamily: 'Lexend',
                fontWeight: FontWeight.w400,
                fontSize: 12,
              ),
            ),
          ),
        ),
      ),
    );
  }

  String _toTitleCase(String text) {
    if (text.isEmpty) return '';
    return text
        .split(RegExp(r'[\s_]+'))
        .map((word) {
          if (word.isEmpty) return '';
          return word[0].toUpperCase() + word.substring(1).toLowerCase();
        })
        .join(' ');
  }

  Widget _buildTabWithIndicator(String tableName) {
    final hasDuplicates = _duplicateRows[tableName]?.isNotEmpty ?? false;
    final hasWarnings = _brokenRelationRows[tableName]?.isNotEmpty ?? false;
    final rowCount = _editableData[tableName]?.length ?? 0;

    Color? indicatorColor;
    if (hasDuplicates) {
      indicatorColor = AppColors.red;
    } else if (hasWarnings) {
      indicatorColor = Colors.orange;
    }

    return Tab(
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text('${_toTitleCase(tableName)} ($rowCount)'),
          if (indicatorColor != null) ...[
            const SizedBox(width: 8),
            CircleAvatar(backgroundColor: indicatorColor, radius: 4),
          ],
        ],
      ),
    );
  }

  Widget _buildInfoAlert({
    required IconData icon,
    required Color color,
    required String title,
    required Widget details,
  }) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: color.withOpacity(0.08),
        border: Border(left: BorderSide(width: 4, color: color)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, color: color, size: 22),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: TextStyle(
                    color: color,
                    fontWeight: FontWeight.w400,
                    fontSize: 14,
                  ),
                ),
                const SizedBox(height: 4),
                details,
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildColumnValidationInfoBox(String tableName) {
    if (!_editableData.containsKey(tableName)) return const SizedBox.shrink();
    final detailsStyle = TextStyle(
      fontSize: 12,
      color: Colors.black.withOpacity(0.8),
      fontWeight: FontWeight.w300,
    );
    if (_editableData[tableName]!.isEmpty) {
      return Container(
        margin: const EdgeInsets.only(bottom: 16),
        child: _buildInfoAlert(
          icon: Icons.check_circle_outlined,
          color: AppColors.schneiderGreen,
          title: "Struktur Kolom Sesuai",
          details: Text(
            "Tidak ada data untuk diimpor di tabel ini.",
            style: detailsStyle,
          ),
        ),
      );
    }
    final expectedColumns = _templateColumns[tableName]?.toSet() ?? <String>{};
    if (expectedColumns.isEmpty) return const SizedBox.shrink();
    final actualColumns = _editableData[tableName]!.first.keys.toSet();
    final missingColumns = expectedColumns.difference(actualColumns).toList();
    final unrecognizedColumns = actualColumns
        .difference(expectedColumns)
        .toList();
    if (missingColumns.isEmpty && unrecognizedColumns.isEmpty) {
      return Container(
        margin: const EdgeInsets.only(bottom: 16),
        child: _buildInfoAlert(
          icon: Icons.check_circle_outlined,
          color: AppColors.schneiderGreen,
          title: "Struktur Kolom Sesuai",
          details: Text(
            "Semua kolom yang diperlukan sudah ada dan dikenali.",
            style: detailsStyle,
          ),
        ),
      );
    }
    return Container(
      margin: const EdgeInsets.only(bottom: 16),
      child: _buildInfoAlert(
        icon: Icons.warning_amber_sharp,
        color: AppColors.orange,
        title: "Struktur Kolom Tidak Sesuai",
        details: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (missingColumns.isNotEmpty) ...[
              const Text(
                "Kolom yang hilang:",
                style: TextStyle(fontWeight: FontWeight.w500),
              ),
              Text(
                "  • ${missingColumns.join('\n  • ').toLowerCase()}",
                style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
              ),
              const Text(
                "Gunakan tombol (+) di header untuk menambahkan.",
                style: TextStyle(fontSize: 11, color: Colors.black54),
              ),
              const SizedBox(height: 8),
            ],
            if (unrecognizedColumns.isNotEmpty) ...[
              const Text(
                "Kolom tidak dikenali:",
                style: TextStyle(fontWeight: FontWeight.w500),
              ),
              Text(
                "  • ${unrecognizedColumns.join('\n  • ').toLowerCase()}",
                style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
              ),
              const Text(
                "Ganti nama atau hapus kolom ini menggunakan menu (⋮) di header.",
                style: TextStyle(fontSize: 11, color: Colors.black54),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildDataTable(String tableName, List<Map<String, dynamic>> rows) {
    final columns = rows.isNotEmpty
        ? rows.first.keys.toList()
        : (_templateColumns[tableName] ?? []);
    return SingleChildScrollView(
      padding: const EdgeInsets.fromLTRB(20, 16, 20, 20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _buildColumnValidationInfoBox(tableName),
          if (columns.isEmpty && rows.isEmpty)
            Center(child: Text('Tidak ada data untuk tabel "$tableName".'))
          else
            Container(
              decoration: BoxDecoration(
                border: Border.all(color: AppColors.grayLight),
                borderRadius: BorderRadius.circular(8),
              ),
              child: ClipRRect(
                borderRadius: BorderRadius.circular(7),
                child: SingleChildScrollView(
                  scrollDirection: Axis.horizontal,
                  child: DataTable(
                    headingRowColor: WidgetStateProperty.all(
                      AppColors.grayLight.withOpacity(0.4),
                    ),
                    headingTextStyle: const TextStyle(
                      fontWeight: FontWeight.w500,
                      fontFamily: 'Lexend',
                      color: AppColors.black,
                      fontSize: 12,
                    ),
                    dataTextStyle: const TextStyle(
                      fontWeight: FontWeight.w300,
                      fontFamily: 'Lexend',
                      color: AppColors.black,
                      fontSize: 12,
                    ),
                    columns: [
                      ...columns.map(
                        (col) => DataColumn(
                          label: _buildColumnHeader(tableName, col),
                        ),
                      ),
                      DataColumn(
                        label: IconButton(
                          icon: const Icon(
                            Icons.add,
                            color: AppColors.schneiderGreen,
                          ),
                          tooltip: 'Tambah Kolom',
                          onPressed: () => _showAddColumnBottomSheet(tableName),
                        ),
                      ),
                      const DataColumn(label: Center(child: Text('Aksi'))),
                    ],
                    rows: List.generate(rows.length, (index) {
                      final rowData = rows[index];
                      final isDuplicate =
                          _duplicateRows[tableName]?.contains(index) ?? false;
                      final isBroken =
                          _brokenRelationRows[tableName]?.contains(index) ??
                          false;
                      return DataRow(
                        color: WidgetStateProperty.resolveWith<Color?>((s) {
                          if (isDuplicate)
                            return AppColors.red.withOpacity(0.1);
                          if (isBroken) return Colors.yellow.withOpacity(0.2);
                          return null;
                        }),
                        cells: [
                          ...columns.map(
                            (colName) => DataCell(
                              SizedBox(
                                width: 180,
                                child: TextFormField(
                                  initialValue:
                                      rowData[colName]?.toString() ?? '',
                                  cursorColor: AppColors.schneiderGreen,
                                  style: const TextStyle(
                                    fontSize: 12,
                                    fontWeight: FontWeight.w300,
                                    fontFamily: 'Lexend',
                                  ),
                                  decoration: const InputDecoration(
                                    isDense: true,
                                    border: InputBorder.none,
                                    focusedBorder: UnderlineInputBorder(
                                      borderSide: BorderSide(
                                        color: AppColors.schneiderGreen,
                                        width: 1.5,
                                      ),
                                    ),
                                    contentPadding: EdgeInsets.symmetric(
                                      vertical: 4,
                                      horizontal: 2,
                                    ),
                                  ),
                                  onChanged: (value) {
                                    rowData[colName] = value;
                                    _revalidateOnDataChange();
                                  },
                                ),
                              ),
                            ),
                          ),
                          const DataCell(SizedBox()),
                          DataCell(
                            Center(
                              child: IconButton(
                                icon: const Icon(
                                  Icons.more_vert,
                                  color: AppColors.gray,
                                  size: 18,
                                ),
                                onPressed: () => _showRowActionsBottomSheet(
                                  tableName,
                                  index,
                                ),
                              ),
                            ),
                          ),
                        ],
                      );
                    }),
                  ),
                ),
              ),
            ),
          const SizedBox(height: 16),
          Align(
            alignment: Alignment.centerRight,
            child: OutlinedButton.icon(
              icon: const Icon(Icons.add_circle_outline, size: 18),
              label: const Text(
                'Tambah Baris',
                style: TextStyle(
                  fontFamily: 'Lexend',
                  fontWeight: FontWeight.w400,
                  fontSize: 12,
                ),
              ),
              onPressed: () => _addRow(tableName),
              style: OutlinedButton.styleFrom(
                foregroundColor: AppColors.schneiderGreen,
                side: BorderSide(color: AppColors.gray.withOpacity(0.5)),
              ),
            ),
          ),
        ],
      ),
    );
  }

  void _showColumnActionsBottomSheet(String tableName, String columnName) {
    showModalBottomSheet(
      context: context,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (context) {
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
              Text(
                'Aksi untuk Kolom "${_toTitleCase(columnName)}"',
                style: const TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.w500,
                ),
              ),
              const SizedBox(height: 16),
              _buildBottomSheetAction(
                icon: Icons.edit_outlined,
                title: 'Ganti Nama Kolom',
                onTap: () {
                  Navigator.pop(context);
                  _showRenameColumnBottomSheet(tableName, columnName);
                },
              ),
              const Divider(height: 1),
              _buildBottomSheetAction(
                icon: Icons.delete_outline,
                title: 'Hapus Kolom',
                isDestructive: true,
                onTap: () {
                  Navigator.pop(context);
                  _showDeleteColumnConfirmationBottomSheet(
                    tableName,
                    columnName,
                  );
                },
              ),
            ],
          ),
        );
      },
    );
  }

  void _showRowActionsBottomSheet(String tableName, int index) {
    final rowData = _editableData[tableName]![index];
    final isDuplicate = _duplicateRows[tableName]?.contains(index) ?? false;
    final isBroken = _brokenRelationRows[tableName]?.contains(index) ?? false;
    final pkColumn = _getPkColumn(tableName);
    final pkValue = (pkColumn.isNotEmpty && rowData.containsKey(pkColumn))
        ? rowData[pkColumn]
        : 'Baris ${index + 1}';
    showModalBottomSheet(
      context: context,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (context) {
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
              Text(
                'Aksi untuk Baris "$pkValue"',
                style: const TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.w500,
                ),
              ),
              const SizedBox(height: 16),
              if (!isBroken && !isDuplicate)
                Text(
                  'Tidak ada masalah pada baris ini.',
                  style: TextStyle(color: AppColors.gray),
                ),
              if (isBroken)
                _buildInfoAlert(
                  icon: Icons.warning_amber_rounded,
                  color: Colors.orange,
                  title: "Peringatan: Relasi Tidak Ditemukan",
                  details: Text(
                    _getBrokenRelationMessage(tableName),
                    style: const TextStyle(fontSize: 12),
                  ),
                ),
              if (isDuplicate) ...[
                if (isBroken) const SizedBox(height: 8),
                _buildInfoAlert(
                  icon: Icons.error_outline,
                  color: AppColors.red,
                  title: "Error: Data Duplikat",
                  details: Text(
                    'Nilai "$pkValue" untuk kolom "$pkColumn" sudah ada dan tidak bisa ditambahkan lagi.',
                    style: const TextStyle(fontSize: 12),
                  ),
                ),
              ],
              const SizedBox(height: 16),
              const Divider(height: 1),
              _buildBottomSheetAction(
                icon: Icons.delete_outline,
                title: 'Hapus Baris',
                isDestructive: true,
                onTap: () {
                  Navigator.pop(context);
                  _deleteRow(tableName, index);
                },
              ),
            ],
          ),
        );
      },
    );
  }

  void _showRenameColumnBottomSheet(String tableName, String oldName) {
    final controller = TextEditingController(text: oldName);
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (context) => Padding(
        padding: EdgeInsets.fromLTRB(
          20,
          16,
          20,
          MediaQuery.of(context).viewInsets.bottom + 16,
        ),
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
              'Ganti Nama Kolom',
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: controller,
              autofocus: true,
              style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w300),
              decoration: InputDecoration(
                hintText: 'Masukkan Nama Kolom Baru',
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
              ),
            ),
            const SizedBox(height: 32),
            _buildActionButtons(
              context: context,
              onSave: () {
                final newName = controller.text.trim().replaceAll(' ', '_');
                _renameColumn(tableName, oldName, newName);
                Navigator.pop(context);
              },
            ),
          ],
        ),
      ),
    );
  }

  void _showAddColumnBottomSheet(String tableName) {
    final controller = TextEditingController();
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (context) => Padding(
        padding: EdgeInsets.fromLTRB(
          20,
          16,
          20,
          MediaQuery.of(context).viewInsets.bottom + 16,
        ),
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
              'Tambah Kolom Baru',
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: controller,
              autofocus: true,
              style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w300),
              decoration: InputDecoration(
                hintText: 'Masukkan Nama Kolom (tanpa spasi)',
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
              ),
            ),
            const SizedBox(height: 32),
            _buildActionButtons(
              context: context,
              saveLabel: "Tambah",
              onSave: () {
                final newName = controller.text.trim().replaceAll(' ', '_');
                _addNewColumn(tableName, newName);
                Navigator.pop(context);
              },
            ),
          ],
        ),
      ),
    );
  }

  void _showDeleteColumnConfirmationBottomSheet(
    String tableName,
    String columnName,
  ) {
    showModalBottomSheet(
      context: context,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (context) => Padding(
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
              'Hapus Kolom?',
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
            ),
            const SizedBox(height: 8),
            Text(
              'Anda yakin ingin menghapus kolom "${_toTitleCase(columnName)}"? Tindakan ini tidak dapat diurungkan.',
              style: const TextStyle(color: AppColors.gray, fontSize: 14),
            ),
            const SizedBox(height: 32),
            _buildActionButtons(
              context: context,
              saveLabel: "Ya, Hapus",
              isDestructive: true,
              onSave: () {
                _deleteColumn(tableName, columnName);
                Navigator.pop(context);
              },
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildActionButtons({
    required BuildContext context,
    required VoidCallback onSave,
    String saveLabel = "Simpan",
    bool isDestructive = false,
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
              backgroundColor: isDestructive
                  ? AppColors.red
                  : AppColors.schneiderGreen,
              foregroundColor: Colors.white,
              elevation: 0,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(6),
              ),
            ),
            child: Text(saveLabel, style: const TextStyle(fontSize: 12)),
          ),
        ),
      ],
    );
  }

  Widget _buildBottomSheetAction({
    required IconData icon,
    required String title,
    required VoidCallback onTap,
    bool isDestructive = false,
  }) {
    final color = isDestructive ? AppColors.red : AppColors.black;
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(8),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 12.0, horizontal: 8.0),
        child: Row(
          children: [
            Icon(icon, color: color),
            const SizedBox(width: 16),
            Text(
              title,
              style: TextStyle(
                color: color,
                fontSize: 14,
                fontWeight: FontWeight.w400,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildColumnHeader(String tableName, String columnName) {
    final expectedColumns = _templateColumns[tableName]?.toSet() ?? <String>{};
    final bool isUnrecognized = !expectedColumns.contains(columnName);

    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Text(_toTitleCase(columnName)),
            Text(
              columnName,
              style: TextStyle(
                color: isUnrecognized ? AppColors.red : AppColors.gray,
                fontSize: 10,
                fontWeight: FontWeight.w300,
              ),
            ),
          ],
        ),
        IconButton(
          padding: EdgeInsets.zero,
          constraints: const BoxConstraints(),
          icon: const Icon(Icons.more_vert, size: 18, color: AppColors.gray),
          onPressed: () => _showColumnActionsBottomSheet(tableName, columnName),
        ),
      ],
    );
  }

  String _getPkColumn(String tableName) {
    const Map<String, String> pkMap = {
      'panels': 'no_pp',
      'companies': 'id',
      'company_accounts': 'username',
    };
    return pkMap[tableName] ?? '';
  }

  String _getBrokenRelationMessage(String tableName) {
    switch (tableName) {
      case 'company_accounts':
        return 'Peringatan: ID Perusahaan untuk akun ini tidak ditemukan. Pastikan data perusahaan juga diimpor atau sudah ada di database.';
      case 'panels':
        return 'Peringatan: ID Pembuat atau Vendor untuk panel ini tidak ditemukan.';
      case 'busbars':
      case 'components':
      case 'palet':
      case 'corepart':
        return 'Peringatan: ID Panel atau Vendor untuk item ini tidak ditemukan.';
      default:
        return 'Peringatan: Terjadi potensi masalah relasi data.';
    }
  }
}
