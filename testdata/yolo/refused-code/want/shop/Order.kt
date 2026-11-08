package shop

import java.util.UUID

data class Order(
    val id: UUID,
    val quantity: Long,
) {
    init {
        require(quantity >= 1) { "quantity must be at least 1" }
    }
}
