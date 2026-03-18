package org.indis.app.data.repository

interface IdentityRepository {
    suspend fun enrollNationalId(nationalId: String): Result<String>
}
